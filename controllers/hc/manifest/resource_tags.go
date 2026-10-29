package manifest

// ResourceTag identifies a GCP Resource Manager tag by its short key and value.
// ParentID is optional; when omitted HyperShift uses the HostedCluster project.
type ResourceTag struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	ParentID string `json:"parentID,omitempty"`
}
