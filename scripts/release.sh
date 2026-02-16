#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'USAGE'
Usage:
  ./scripts/release.sh patch
  ./scripts/release.sh minor
  ./scripts/release.sh major
  ./scripts/release.sh <version>

Examples:
  ./scripts/release.sh patch
  ./scripts/release.sh 0.2.0
USAGE
}

bump="${1:-}"

if [[ -z "${bump}" || "${bump}" == "--help" || "${bump}" == "-h" ]]; then
  usage
  exit 0
fi

if [[ $# -gt 1 ]]; then
  echo "Error: expected a single bump argument." >&2
  usage
  exit 1
fi

if [[ ! "${bump}" =~ ^(patch|minor|major|[0-9]+\.[0-9]+\.[0-9]+)$ ]]; then
  echo "Error: invalid bump argument '${bump}'. Use patch|minor|major|x.y.z." >&2
  exit 1
fi

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_root}"

if [[ -n "$(git status --porcelain)" ]]; then
  echo "Error: working tree is not clean. Commit or stash changes first." >&2
  exit 1
fi

current_branch="$(git rev-parse --abbrev-ref HEAD)"
if [[ "${current_branch}" != "main" ]]; then
  echo "Error: release must be run from main branch. Current branch: ${current_branch}" >&2
  exit 1
fi

echo "Syncing main with origin/main..."
git fetch origin
if ! git pull --ff-only origin main; then
  echo "Error: failed to fast-forward main. Please resolve and retry." >&2
  exit 1
fi

latest_tag="$(git tag --list 'v*' --sort=-version:refname | head -n 1)"
if [[ -z "${latest_tag}" ]]; then
  latest_tag="v0.0.0"
fi

latest_version="${latest_tag#v}"
IFS='.' read -r major minor patch <<< "${latest_version}"

next_version=""
case "${bump}" in
  patch)
    next_version="${major}.${minor}.$((patch + 1))"
    ;;
  minor)
    next_version="${major}.$((minor + 1)).0"
    ;;
  major)
    next_version="$((major + 1)).0.0"
    ;;
  *)
    next_version="${bump}"
    ;;
esac

new_tag="v${next_version}"

if git rev-parse -q --verify "refs/tags/${new_tag}" >/dev/null; then
  echo "Error: tag ${new_tag} already exists." >&2
  exit 1
fi

echo "Running validation..."
go build ./...
go vet ./...
go test ./...

echo "Pushing main..."
git push origin main

echo "Creating and pushing tag ${new_tag}..."
git tag "${new_tag}"
git push origin "${new_tag}"

echo "Release started for ${new_tag}"
echo "GitHub Actions will build assets, publish release files, and update Formula/lunchmoney-cli.rb."
