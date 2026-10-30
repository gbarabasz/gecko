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
disables platform-configured tags. If both API-request and configured tags are
absent, the manifest omits `spec.platform.gcp.resourceTags`
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

## API-request tags and merging

Customers can pass tags in a Cluster POST request, for example this fragment
of an otherwise complete request:

```yaml
spec:
  platform:
    gcp:
      resourceTags:
        - key: environment
          value: production
          parentID: "123456789"
```

These tags are stored in the Gecko Cluster. On every reconciliation, HC reads
that Cluster and merges `spec.platform.gcp.resourceTags` with tags loaded from
the configuration file. Configured tags take precedence on duplicate short keys,
including replacing the requested parentID. The merged list is sorted by key for
stable manifests. At most 50 distinct keys are allowed in the combined result;
HC rejects an oversized union before submitting any manifests.

The public and private API schemas make the Cluster's `resourceTags` immutable:
after creation, PUT/PATCH cannot add, remove, or change them. Tags use the GCP
Resource Manager short-key/value format; parentID must identify an organization
or the Cluster's GCP project. API omission remains valid for existing Clusters.

This validation does **not** freeze the controller's configuration file. HC
always emits the merged desired tags, with no special Firestore handling and no
create/update distinction. Keep the configured tags stable for existing
HostedClusters: changing/removing them or enabling tags for an existing untagged
HostedCluster can produce an immutable-field rejection from HyperShift. Such a
rejection fails the whole HostedCluster apply and can block unrelated updates
until the desired tags are restored. Updating the ConfigMap is therefore not a
supported mechanism to retrofit or change tags on existing HostedClusters.

The per-cluster resource-label propagation gap is separate: this change does not
alter the existing config-only behavior of `customerLabels`.

Run rendering coverage with `go test ./cmd/hc` from `controllers/` (requires Helm).
