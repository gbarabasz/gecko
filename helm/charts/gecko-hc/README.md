# GCP resource labels and tags

Both settings are JSON-encoded strings supplied by the region ArgoCD Application:

```yaml
customerLabels: '{"goog-partner-solution":"example-partner"}'
resourceTags: '[{"key":"redhat-managed","value":"true"},{"key":"environment","value":"production","parentID":"123456789"}]'
```

`customerLabels` comes from `meta_customer_labels`; `resourceTags` comes from
`meta_resource_tags` in the infrastructure-managed `argocd-cluster` metadata.
Tags use the HyperShift GCP resource-tag API: short `key`, `value`, and optional
`parentID` (for example, an organization ID). When omitted, HyperShift uses the
HostedCluster's GCP project as the parent. An empty list (`"[]"`, the default)
disables resource tags. Empty configuration omits `spec.platform.gcp.resourceTags`
rather than sending an empty field. A supporting HyperShift operator/CRD, tag
keys/values, and tag-use IAM permissions must be installed before enabling tags.

## ConfigMap layout

One `<release>-resource-config` ConfigMap contains separate `customer-labels.json`
and `resource-tags.json` keys, only for the configured settings. It is mounted
read-only at `/etc/config/resources` in the `hc` container. The matching
`--customer-labels-file` and `--resource-tags-file` flags tell the application
which files to load. No ConfigMap, volume, or file flags are emitted when both
settings are empty. The pod-template checksum restarts the controller when
configuration changes; files are loaded and validated at startup, not hot-reloaded.

One map is preferable here because both files have the same consumer, ownership,
access requirements, and deployment lifecycle. It needs just one volume, mount,
and rollout checksum. Separate maps would be useful for independent ownership or
access policies, but offer no such benefit here. Separate JSON keys preserve the
different data shapes without changing the existing label value contract.

## Creation-only tags

Resource tags are immutable after HostedCluster creation. Gecko freezes them
(including their absence) in the **first persisted HostedCluster apply desire**.
Later reconciliations atomically preserve that field from the existing desire,
while still updating mutable fields such as the release image and labels. This
also works across controller restarts, retries, overlapping replicas, and
configuration changes/removal. Existing managed HostedClusters retain their
original tags; enabling tags does not retrofit untagged HostedClusters.

The persisted apply desire is the source of this creation snapshot; do not delete
or edit it to change tags on an existing HostedCluster. This is not an adoption
mechanism for out-of-band HostedClusters with no Gecko apply desire. Tag changes
require a new HostedCluster. Configuration changes apply to new creation requests
only, not to requests already submitted but still awaiting creation.

Run rendering coverage with `go test ./cmd/hc` from `controllers/` (requires Helm).
The Firestore creation-only integration tests require `FIRESTORE_EMULATOR_HOST`.
