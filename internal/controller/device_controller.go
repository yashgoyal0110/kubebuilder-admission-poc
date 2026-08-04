package controller

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	devicesv1alpha1 "github.com/yashgoyal0110/kubebuilder-admission-poc/api/v1alpha1"
)

// DeviceReconciler keeps Device status in step with its spec.
//
// The reconcile logic here is deliberately thin. The point of this project is that this controller
// and the admission webhooks share one Manager, not that the reconcile loop is clever.
type DeviceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=devices.yashgoyal.dev,resources=devices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=devices.yashgoyal.dev,resources=devices/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=devices.yashgoyal.dev,resources=devices/finalizers,verbs=update

func (r *DeviceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var device devicesv1alpha1.Device
	if err := r.Get(ctx, req.NamespacedName, &device); err != nil {
		// Not found is normal: the object was deleted between the event and this read.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if device.Status.ObservedGeneration == device.Generation {
		return ctrl.Result{}, nil
	}

	logger.Info("reconciling device", "model", device.Spec.Model, "node", device.Spec.NodeName)

	device.Status.ObservedGeneration = device.Generation
	meta := metav1.Condition{
		Type:               devicesv1alpha1.ConditionReady,
		Status:             metav1.ConditionTrue,
		Reason:             "SpecAccepted",
		Message:            "device spec has been accepted",
		ObservedGeneration: device.Generation,
	}
	setCondition(&device.Status.Conditions, meta)

	if err := r.Status().Update(ctx, &device); err != nil {
		// A conflict means someone else wrote first. Requeue rather than log an error, because
		// the next pass reads the newer object and does the right thing.
		if apierrors.IsConflict(err) {
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func setCondition(conditions *[]metav1.Condition, next metav1.Condition) {
	for i, existing := range *conditions {
		if existing.Type != next.Type {
			continue
		}
		// Only move LastTransitionTime when the status actually changed, otherwise every
		// reconcile rewrites the timestamp and the field stops meaning anything.
		if existing.Status == next.Status {
			next.LastTransitionTime = existing.LastTransitionTime
		} else {
			next.LastTransitionTime = metav1.Now()
		}
		(*conditions)[i] = next
		return
	}
	next.LastTransitionTime = metav1.Now()
	*conditions = append(*conditions, next)
}

func (r *DeviceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&devicesv1alpha1.Device{}).
		Named("device").
		Complete(r)
}
