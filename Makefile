SHELL := /usr/bin/env bash
LOCALBIN ?= $(shell pwd)/bin

# Tool versions are pinned here and installed into ./bin. Deliberately not "use whatever
# controller-gen is on your PATH": that is how a repo ends up with generated files that differ
# between one contributor and the next, and between a laptop and CI.
CONTROLLER_TOOLS_VERSION ?= v0.17.3
ENVTEST_VERSION          ?= release-0.20
ENVTEST_K8S_VERSION      ?= 1.32.0
GOVULNCHECK_VERSION      ?= latest

CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST        ?= $(LOCALBIN)/setup-envtest
GOVULNCHECK    ?= $(LOCALBIN)/govulncheck

.PHONY: all
all: verify-go-version generate manifests fmt vet test

$(LOCALBIN):
	mkdir -p $(LOCALBIN)

$(CONTROLLER_GEN): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

$(ENVTEST): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@$(ENVTEST_VERSION)

$(GOVULNCHECK): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)

.PHONY: generate
generate: $(CONTROLLER_GEN) ## DeepCopy methods
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./api/..."

.PHONY: manifests
manifests: $(CONTROLLER_GEN) ## CRDs, RBAC and webhook configuration, all from markers
	$(CONTROLLER_GEN) crd rbac:roleName=manager-role webhook \
		paths="./..." \
		output:crd:artifacts:config=config/crd \
		output:rbac:artifacts:config=config/rbac \
		output:webhook:artifacts:config=config/webhook

.PHONY: verify-go-version
verify-go-version:
	./hack/verify-go-version.sh

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: test
test: $(ENVTEST)
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" \
		go test ./... -coverprofile cover.out

.PHONY: vulncheck
vulncheck: $(GOVULNCHECK)
	$(GOVULNCHECK) ./...

.PHONY: build
build:
	go build -o $(LOCALBIN)/manager ./cmd

.PHONY: verify
verify: generate manifests ## fail if generated files are stale
	@if [ -n "$$(git status --porcelain api config)" ]; then \
		echo "generated files are out of date, run 'make generate manifests'"; \
		git --no-pager diff -- api config; \
		exit 1; \
	fi

.PHONY: clean
clean:
	rm -rf $(LOCALBIN) cover.out
