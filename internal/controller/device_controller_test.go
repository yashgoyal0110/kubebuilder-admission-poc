package controller

import (
	"context"
	"testing"
	"time"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	devicesv1alpha1 "github.com/yashgoyal0110/kubebuilder-admission-poc/api/v1alpha1"
)

func TestReconcileMarksDeviceReady(t *testing.T) {
	startManager(t)
	ctx := context.Background()

	device := newDevice("ready-device")
	if err := k8sClient.Create(ctx, device); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	t.Cleanup(func() { _ = k8sClient.Delete(ctx, device) })

	var fetched devicesv1alpha1.Device
	eventually(t, 10*time.Second, func() bool {
		if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(device), &fetched); err != nil {
			return false
		}
		return apimeta.IsStatusConditionTrue(fetched.Status.Conditions, devicesv1alpha1.ConditionReady)
	})

	if fetched.Status.ObservedGeneration != fetched.Generation {
		t.Fatalf("observedGeneration %d, want %d",
			fetched.Status.ObservedGeneration, fetched.Generation)
	}
}

func TestConditionTimestampOnlyMovesOnChange(t *testing.T) {
	// Rewriting LastTransitionTime on every pass makes the field useless for working out when
	// something actually broke, and it churns the object for no reason.
	conditions := []metav1.Condition{}
	first := metav1.Condition{Type: devicesv1alpha1.ConditionReady, Status: metav1.ConditionTrue, Reason: "A"}
	setCondition(&conditions, first)
	stamp := conditions[0].LastTransitionTime

	time.Sleep(10 * time.Millisecond)
	setCondition(&conditions, metav1.Condition{
		Type: devicesv1alpha1.ConditionReady, Status: metav1.ConditionTrue, Reason: "B",
	})
	if !conditions[0].LastTransitionTime.Equal(&stamp) {
		t.Fatal("timestamp moved even though status did not change")
	}
	if conditions[0].Reason != "B" {
		t.Fatalf("reason not updated, got %q", conditions[0].Reason)
	}

	setCondition(&conditions, metav1.Condition{
		Type: devicesv1alpha1.ConditionReady, Status: metav1.ConditionFalse, Reason: "C",
	})
	if conditions[0].LastTransitionTime.Equal(&stamp) {
		t.Fatal("timestamp did not move when status changed")
	}
	if len(conditions) != 1 {
		t.Fatalf("expected the condition to be replaced, got %d entries", len(conditions))
	}
}
