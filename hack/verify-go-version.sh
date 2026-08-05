#!/usr/bin/env bash
# Fails if the Go version is not the same everywhere.
#
# Drift here is quiet and expensive: go.mod says one thing, the Dockerfile builds with another,
# CI tests on a third, and a security fix that landed in a patch release is silently missing from
# whichever one you forgot. Cheaper to make the build fail.
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

expected="$(tr -d '[:space:]' < .go-version)"
status=0

check() {
  local file="$1" found="$2"
  if [[ -z "${found}" ]]; then
    echo "  ?  ${file}: no Go version found"
    status=1
  elif [[ "${found}" != "${expected}" ]]; then
    echo "  x  ${file}: ${found}"
    status=1
  else
    echo "  ok ${file}: ${found}"
  fi
}

echo "expecting Go ${expected} (from .go-version)"

check "go.mod" "$(awk '/^go /{print $2; exit}' go.mod)"
check "Dockerfile" "$(sed -n 's/^FROM.*golang:\([0-9.]*\).*/\1/p' Dockerfile | head -1)"

# CI deliberately has no version of its own to check. The workflow uses
# go-version-file: go.mod, so there is one fewer place for this to drift.
if grep -q "go-version:" .github/workflows/*.yml 2>/dev/null; then
  echo "  x  workflows: pin Go with go-version-file: go.mod instead of a literal"
  status=1
fi

exit ${status}
