#!/usr/bin/env python3
"""Update the one production Kustomize image after an image has passed CI."""

import argparse
from pathlib import Path
import re

parser = argparse.ArgumentParser()
parser.add_argument("--repository-url", required=True)
parser.add_argument("--tag", required=True)
args = parser.parse_args()

if not re.fullmatch(r"[0-9a-f]{40}", args.tag):
    parser.error("tag must be a full Git commit SHA")
if not re.fullmatch(r"[0-9]+\.dkr\.ecr\.[a-z0-9-]+\.amazonaws\.com/[a-z0-9/_-]+", args.repository_url):
    parser.error("repository-url must be an ECR repository URL")

path = Path("k8s/overlays/prod/kustomization.yaml")
data = path.read_text()
for key, value in (("newName", args.repository_url), ("newTag", args.tag)):
    data, count = re.subn(rf"(?m)^    {key}: .+$", f"    {key}: {value}", data)
    if count != 1:
        raise SystemExit(f"expected exactly one {key} entry, found {count}")
path.write_text(data)
