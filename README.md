# GECKO

<p align="center">
	<img src="docs/logos/gecko-logo.png" width=400px>
</p>

---

GCP Engine for Cluster Kubernetes Orchestration. Bringing the HyperShift project to Google Cloud Platform.

---

## Running pull request builds

Konflux publishes the images built for every pull request against `main` to a
development Artifact Registry repository, tagged with the PR's head commit SHA
(full and 7-character short):

```
northamerica-northeast2-docker.pkg.dev/gcp-hcp-commons-dev/gcp-hcp-pr-images/gecko-controllers:<sha>
northamerica-northeast2-docker.pkg.dev/gcp-hcp-commons-dev/gcp-hcp-pr-images/gecko-platform-api-server:<sha>
```

The optional `<component>-publish-pr-images` check on the PR turns green once
the images are pushed. Dev-all-in-one and platform-ci e2e environments can pull
them directly; members of `gcp-hcp-eng@redhat.com` can pull them with their own
gcloud credentials. Images are deleted after 14 days; push to the PR or re-run
the check to publish them again.
