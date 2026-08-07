package v1alpha1

import (
	"fmt"
	"net/url"
	"slices"

	"k8s.io/apimachinery/pkg/util/validation/field"

	devicesv1alpha1 "github.com/yashgoyal0110/kubebuilder-admission-poc/api/v1alpha1"
)

// Each protocol dials a different kind of address, so "is this a URL" is not a useful check on its
// own. An opcua device pointed at an mqtt endpoint is accepted by any generic URL validation and
// then fails at the edge, where it is someone else's afternoon.
var schemesByProtocol = map[devicesv1alpha1.Protocol][]string{
	devicesv1alpha1.ProtocolMQTT:      {"tcp", "ssl", "mqtt", "mqtts"},
	devicesv1alpha1.ProtocolModbus:    {"tcp", "rtu"},
	devicesv1alpha1.ProtocolOPCUA:     {"opc.tcp"},
	devicesv1alpha1.ProtocolBluetooth: {"ble"},
}

func validateSpec(device *devicesv1alpha1.Device) field.ErrorList {
	var errs field.ErrorList
	endpointPath := field.NewPath("spec", "endpoint")

	parsed, err := url.Parse(device.Spec.Endpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return append(errs, field.Invalid(endpointPath, device.Spec.Endpoint,
			"must be an absolute URL with a scheme and a host"))
	}

	// Protocol may be empty here when validation runs on an object the defaulter did not touch,
	// so treat empty as the default rather than as an unknown protocol.
	protocol := device.Spec.Protocol
	if protocol == "" {
		protocol = devicesv1alpha1.ProtocolMQTT
	}

	allowed, known := schemesByProtocol[protocol]
	if !known {
		// The CRD enum should have caught this already. If it somehow did not, refuse rather
		// than let an unvalidated endpoint through.
		return append(errs, field.Invalid(field.NewPath("spec", "protocol"), protocol,
			"unsupported protocol"))
	}

	if !slices.Contains(allowed, parsed.Scheme) {
		errs = append(errs, field.Invalid(endpointPath, device.Spec.Endpoint,
			fmt.Sprintf("scheme %q is not valid for protocol %q, expected one of %v",
				parsed.Scheme, protocol, allowed)))
	}

	return errs
}
