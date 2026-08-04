// Package v1alpha1 holds the admission logic for Device.
//
// These are plain controller-runtime interfaces (CustomDefaulter, CustomValidator). They are
// registered on the same Manager as the controller, so they inherit its client, scheme, cache,
// logger, health checks, metrics and leader election, and they are served by the Manager's webhook
// server rather than an http.Server of their own.
package v1alpha1

import (
	"context"
	"fmt"
	"net/url"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	devicesv1alpha1 "github.com/yashgoyal0110/kubebuilder-admission-poc/api/v1alpha1"
)

var devicelog = logf.Log.WithName("device-webhook")

// SetupDeviceWebhookWithManager registers both webhooks against the Manager.
func SetupDeviceWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&devicesv1alpha1.Device{}).
		WithDefaulter(&DeviceCustomDefaulter{}).
		WithValidator(&DeviceCustomValidator{}).
		Complete()
}

// The webhook manifests come from these markers. Nothing registers a
// ValidatingWebhookConfiguration in Go at runtime, which means what the cluster enforces is
// reviewable in a YAML diff instead of only visible once the process is running.

// +kubebuilder:webhook:path=/mutate-devices-yashgoyal-dev-v1alpha1-device,mutating=true,failurePolicy=fail,sideEffects=None,groups=devices.yashgoyal.dev,resources=devices,verbs=create;update,versions=v1alpha1,name=mdevice.kb.io,admissionReviewVersions=v1

// DeviceCustomDefaulter fills in what the user left out.
type DeviceCustomDefaulter struct{}

var _ webhook.CustomDefaulter = &DeviceCustomDefaulter{}

func (d *DeviceCustomDefaulter) Default(_ context.Context, obj runtime.Object) error {
	device, ok := obj.(*devicesv1alpha1.Device)
	if !ok {
		return fmt.Errorf("expected a Device, got %T", obj)
	}

	if device.Spec.Protocol == "" {
		device.Spec.Protocol = devicesv1alpha1.ProtocolMQTT
	}
	if device.Spec.OfflineMigration == nil {
		enabled := true
		device.Spec.OfflineMigration = &enabled
	}

	devicelog.V(1).Info("defaulted device", "name", device.Name, "protocol", device.Spec.Protocol)
	return nil
}

// +kubebuilder:webhook:path=/validate-devices-yashgoyal-dev-v1alpha1-device,mutating=false,failurePolicy=fail,sideEffects=None,groups=devices.yashgoyal.dev,resources=devices,verbs=create;update,versions=v1alpha1,name=vdevice.kb.io,admissionReviewVersions=v1

// DeviceCustomValidator rejects what the CRD schema can't express on its own.
type DeviceCustomValidator struct{}

var _ webhook.CustomValidator = &DeviceCustomValidator{}

func (v *DeviceCustomValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	device, ok := obj.(*devicesv1alpha1.Device)
	if !ok {
		return nil, fmt.Errorf("expected a Device, got %T", obj)
	}
	return nil, validateSpec(device).ToAggregate()
}

func (v *DeviceCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	oldDevice, ok := oldObj.(*devicesv1alpha1.Device)
	if !ok {
		return nil, fmt.Errorf("expected a Device, got %T", oldObj)
	}
	newDevice, ok := newObj.(*devicesv1alpha1.Device)
	if !ok {
		return nil, fmt.Errorf("expected a Device, got %T", newObj)
	}

	errs := validateSpec(newDevice)

	// Immutability is the reason this webhook exists at all. A CRD schema can describe what a
	// valid Device looks like, but it cannot see the previous object, so it cannot say "that
	// field was allowed to be anything at creation and nothing afterwards".
	if oldDevice.Spec.Model != newDevice.Spec.Model {
		errs = append(errs, field.Invalid(
			field.NewPath("spec", "model"), newDevice.Spec.Model,
			fmt.Sprintf("field is immutable, was %q", oldDevice.Spec.Model)))
	}

	return nil, errs.ToAggregate()
}

func (v *DeviceCustomValidator) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func validateSpec(device *devicesv1alpha1.Device) field.ErrorList {
	var errs field.ErrorList
	specPath := field.NewPath("spec")

	parsed, err := url.Parse(device.Spec.Endpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		errs = append(errs, field.Invalid(specPath.Child("endpoint"), device.Spec.Endpoint,
			"must be an absolute URL with a scheme and a host"))
	}

	return errs
}
