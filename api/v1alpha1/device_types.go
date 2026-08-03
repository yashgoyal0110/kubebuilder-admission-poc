package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Protocol is how the edge node talks to the device.
// +kubebuilder:validation:Enum=mqtt;modbus;opcua;bluetooth
type Protocol string

const (
	ProtocolMQTT      Protocol = "mqtt"
	ProtocolModbus    Protocol = "modbus"
	ProtocolOPCUA     Protocol = "opcua"
	ProtocolBluetooth Protocol = "bluetooth"
)

// DeviceSpec describes a physical device attached to an edge node.
type DeviceSpec struct {
	// Model is the DeviceModel this device conforms to. Immutable after creation: changing the
	// model of a live device would reinterpret every property it has already reported.
	// +kubebuilder:validation:MinLength=1
	Model string `json:"model"`

	// Protocol defaults to mqtt when not set.
	// +optional
	Protocol Protocol `json:"protocol,omitempty"`

	// Endpoint the edge node dials to reach the device.
	// +kubebuilder:validation:MinLength=1
	Endpoint string `json:"endpoint"`

	// NodeName is the edge node this device is attached to.
	// +kubebuilder:validation:MinLength=1
	NodeName string `json:"nodeName"`

	// OfflineMigration allows the device to be re-homed to another node when its current node
	// goes offline.
	// +optional
	OfflineMigration *bool `json:"offlineMigration,omitempty"`
}

// DeviceStatus is the observed state of a Device.
type DeviceStatus struct {
	// ObservedGeneration is the .metadata.generation the controller last acted on.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions follow the standard Kubernetes condition contract.
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// Condition types set by the controller.
const (
	// ConditionReady is true once the device spec has been accepted and reconciled.
	ConditionReady = "Ready"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=dev
// +kubebuilder:printcolumn:name="Model",type=string,JSONPath=`.spec.model`
// +kubebuilder:printcolumn:name="Protocol",type=string,JSONPath=`.spec.protocol`
// +kubebuilder:printcolumn:name="Node",type=string,JSONPath=`.spec.nodeName`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Device is an edge device managed through the cloud API.
type Device struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeviceSpec   `json:"spec,omitempty"`
	Status DeviceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DeviceList contains a list of Device.
type DeviceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Device `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Device{}, &DeviceList{})
}
