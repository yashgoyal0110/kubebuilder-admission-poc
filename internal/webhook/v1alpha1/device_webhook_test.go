package v1alpha1

import (
	"context"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	devicesv1alpha1 "github.com/yashgoyal0110/kubebuilder-admission-poc/api/v1alpha1"
)

func device(name string) *devicesv1alpha1.Device {
	return &devicesv1alpha1.Device{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
		Spec: devicesv1alpha1.DeviceSpec{
			Model:    "thermostat-v1",
			Endpoint: "tcp://192.168.1.40:1883",
			NodeName: "edge-node-1",
		},
	}
}

func TestDefaultingFillsProtocolAndOfflineMigration(t *testing.T) {
	ctx := context.Background()
	d := device("defaulted")

	if err := k8sClient.Create(ctx, d); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	t.Cleanup(func() { _ = k8sClient.Delete(ctx, d) })

	var got devicesv1alpha1.Device
	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(d), &got); err != nil {
		t.Fatalf("get failed: %v", err)
	}

	if got.Spec.Protocol != devicesv1alpha1.ProtocolMQTT {
		t.Errorf("protocol = %q, want mqtt", got.Spec.Protocol)
	}
	if got.Spec.OfflineMigration == nil || !*got.Spec.OfflineMigration {
		t.Error("offlineMigration should default to true")
	}
}

func TestValidationRejectsANonAbsoluteEndpoint(t *testing.T) {
	ctx := context.Background()
	d := device("bad-endpoint")
	d.Spec.Endpoint = "192.168.1.40"

	err := k8sClient.Create(ctx, d)
	if err == nil {
		t.Cleanup(func() { _ = k8sClient.Delete(ctx, d) })
		t.Fatal("expected the webhook to reject an endpoint with no scheme")
	}
	if !strings.Contains(err.Error(), "absolute URL") {
		t.Fatalf("rejected for the wrong reason: %v", err)
	}
}

func TestModelIsImmutable(t *testing.T) {
	ctx := context.Background()
	d := device("immutable-model")

	if err := k8sClient.Create(ctx, d); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	t.Cleanup(func() { _ = k8sClient.Delete(ctx, d) })

	var got devicesv1alpha1.Device
	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(d), &got); err != nil {
		t.Fatalf("get failed: %v", err)
	}

	got.Spec.Model = "thermostat-v2"
	err := k8sClient.Update(ctx, &got)
	if err == nil {
		t.Fatal("expected the webhook to reject a model change")
	}
	if !strings.Contains(err.Error(), "immutable") {
		t.Fatalf("rejected for the wrong reason: %v", err)
	}
}

func TestUnchangedModelStillUpdates(t *testing.T) {
	// The immutability check must compare old against new, not simply reject every update.
	ctx := context.Background()
	d := device("mutable-endpoint")

	if err := k8sClient.Create(ctx, d); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	t.Cleanup(func() { _ = k8sClient.Delete(ctx, d) })

	var got devicesv1alpha1.Device
	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(d), &got); err != nil {
		t.Fatalf("get failed: %v", err)
	}

	got.Spec.Endpoint = "tcp://192.168.1.41:1883"
	if err := k8sClient.Update(ctx, &got); err != nil {
		t.Fatalf("update with an unchanged model should be allowed: %v", err)
	}
}

func TestSchemaRejectsAnUnknownProtocol(t *testing.T) {
	// This one is enforced by the CRD enum rather than the webhook. Worth pinning so that if the
	// marker is ever dropped, a test fails instead of the API server quietly accepting junk.
	ctx := context.Background()
	d := device("bad-protocol")
	d.Spec.Protocol = devicesv1alpha1.Protocol("carrier-pigeon")

	if err := k8sClient.Create(ctx, d); err == nil {
		t.Cleanup(func() { _ = k8sClient.Delete(ctx, d) })
		t.Fatal("expected the CRD schema to reject an unsupported protocol")
	}
}
