package controller

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	devicesv1alpha1 "github.com/yashgoyal0110/kubebuilder-admission-poc/api/v1alpha1"
)

var (
	testEnv   *envtest.Environment
	cfg       *rest.Config
	k8sClient client.Client
	scheme    = runtime.NewScheme()
)

// TestMain brings up a real API server and etcd through envtest, applies the generated CRD, and
// runs every test in this package against it. Fake clients accept anything the Go types allow,
// which is exactly the wrong thing to test admission against.
func TestMain(m *testing.M) {
	logf.SetLogger(zap.New(zap.WriteTo(os.Stderr), zap.UseDevMode(true)))

	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(devicesv1alpha1.AddToScheme(scheme))

	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	cfg, err = testEnv.Start()
	if err != nil {
		panic("failed to start envtest, run 'make test' so KUBEBUILDER_ASSETS is set: " + err.Error())
	}

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		panic(err)
	}

	code := m.Run()

	if err := testEnv.Stop(); err != nil {
		panic(err)
	}
	os.Exit(code)
}

// startManager runs a Manager with the reconciler attached, for tests that need reconciliation to
// actually happen rather than being invoked directly.
func startManager(t *testing.T) {
	t.Helper()

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:  scheme,
		Metrics: metricsserver.Options{BindAddress: "0"},
	})
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	if err := (&DeviceReconciler{Client: mgr.GetClient(), Scheme: mgr.GetScheme()}).
		SetupWithManager(mgr); err != nil {
		t.Fatalf("failed to set up reconciler: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	go func() {
		if err := mgr.Start(ctx); err != nil {
			panic(err)
		}
	}()

	if !mgr.GetCache().WaitForCacheSync(ctx) {
		t.Fatal("cache did not sync")
	}
}

// newDevice builds a Device that passes validation. The webhooks are not installed in this
// suite, so these tests exercise reconciliation on its own.
func newDevice(name string) *devicesv1alpha1.Device {
	return &devicesv1alpha1.Device{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
		Spec: devicesv1alpha1.DeviceSpec{
			Model:    "thermostat-v1",
			Endpoint: "tcp://192.168.1.40:1883",
			NodeName: "edge-node-1",
			Protocol: devicesv1alpha1.ProtocolMQTT,
		},
	}
}

func eventually(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("condition not met within %s", timeout)
}
