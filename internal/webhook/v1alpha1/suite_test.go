package v1alpha1

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	devicesv1alpha1 "github.com/yashgoyal0110/kubebuilder-admission-poc/api/v1alpha1"
)

var (
	testEnv   *envtest.Environment
	k8sClient client.Client
	scheme    = runtime.NewScheme()
)

// TestMain starts an API server that is actually configured to call these webhooks, using the
// same generated manifests that would be applied to a real cluster.
//
// This is the part worth having. Calling Default() and ValidateCreate() directly in a unit test
// proves the functions work. It does not prove the webhook configuration points at the right
// path, matches the right resources, or that the API server can reach the server at all, which is
// where this kind of migration usually goes wrong.
func TestMain(m *testing.M) {
	logf.SetLogger(zap.New(zap.WriteTo(os.Stderr), zap.UseDevMode(true)))

	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(devicesv1alpha1.AddToScheme(scheme))

	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "config", "crd")},
		ErrorIfCRDPathMissing: true,
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths: []string{filepath.Join("..", "..", "..", "config", "webhook")},
		},
	}

	cfg, err := testEnv.Start()
	if err != nil {
		panic("failed to start envtest, run 'make test' so KUBEBUILDER_ASSETS is set: " + err.Error())
	}

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		panic(err)
	}

	opts := testEnv.WebhookInstallOptions
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:  scheme,
		Metrics: metricsserver.Options{BindAddress: "0"},
		WebhookServer: webhook.NewServer(webhook.Options{
			Host:    opts.LocalServingHost,
			Port:    opts.LocalServingPort,
			CertDir: opts.LocalServingCertDir,
			TLSOpts: []func(*tls.Config){func(c *tls.Config) { c.MinVersion = tls.VersionTLS12 }},
		}),
	})
	if err != nil {
		panic(err)
	}

	if err := SetupDeviceWebhookWithManager(mgr); err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		if err := mgr.Start(ctx); err != nil {
			panic(err)
		}
	}()

	waitForWebhookServer(opts.LocalServingHost, opts.LocalServingPort)

	code := m.Run()

	cancel()
	if err := testEnv.Stop(); err != nil {
		panic(err)
	}
	os.Exit(code)
}

// waitForWebhookServer blocks until the TLS listener accepts a connection. Without this the first
// test races the server's startup and fails with a connection refused that looks like a webhook
// bug rather than a timing one.
func waitForWebhookServer(host string, port int) {
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := tls.DialWithDialer(
			&net.Dialer{Timeout: time.Second}, "tcp", addr,
			&tls.Config{InsecureSkipVerify: true}, //nolint:gosec // self-signed envtest certs
		)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	panic("webhook server did not come up within 30s")
}
