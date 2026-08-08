# What this maps onto

This repository is a scaled-down version of one change: moving admission webhooks out of a separate
service and into the controller-runtime Manager that already runs the reconcilers.

## The two shapes

**Separate admission service.** A second binary, a second Deployment, a second Service, its own
`http.Server` and TLS setup, a `ValidatingWebhookConfiguration` built in Go and pushed to the API
server at startup, and certificates minted by a shell script at install time.

**One Manager.** The webhook server is a field on `ctrl.Options`. Handlers implement
`CustomDefaulter` and `CustomValidator`. Webhook configuration is generated from
`+kubebuilder:webhook` markers into YAML. Certificates come from cert-manager, and the CA bundle is
injected by annotation.

What the second shape gets for free, because the Manager already had it: one client, one cache, one
scheme, one logger, one set of health and readiness probes, one metrics endpoint, leader election,
and a single shutdown path.

## The parts that are easy to get wrong

**Readiness.** The pod must not report ready until the webhook server can serve. With
`failurePolicy: Fail`, a pod that is ready-but-not-serving makes the API server reject every write
it is registered for. `mgr.GetWebhookServer().StartedChecker()` is wired into `AddReadyzCheck` for
exactly this.

**Leader election.** Reconciliation should run in one replica. Admission must run in *all* of them,
because every pod sits behind the Service. Putting the webhook server on the Manager gets this
right by default, since webhook serving is not gated on leadership, but it is worth knowing that is
a decision the framework already made and not an accident.

**Ordering during a migration.** For a while both the old and new webhook configurations exist and
point at different Services. Whichever one the API server calls has to work. The safe sequence is:
stand the new path up, prove parity, repoint the configuration, then delete the old Deployment.
Deleting first means a window where nothing serves and everything fails closed.

**Certificates.** A self-signed CA with a ten year lifetime generated at install time is a
certificate nobody rotates, because nothing ever reminds them to. cert-manager renews at 15 days
before a 90 day expiry, and re-injects the CA bundle when it does.

## What is deliberately not here

One CRD instead of six. No conversion webhooks. No upgrade path from an existing deployment, which
in a real migration is most of the work. The point was to build the end state and find out where it
is sharp, not to pretend the migration itself is small.
