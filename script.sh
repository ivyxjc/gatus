#!/bin/bash
set -eo pipefail
echo "+++++++++++++++++++++++++++++++++++++"

export $(cat /aws.env | xargs)
env > /aws.env

BUCKET="simcloud-aws-props-dev"
KEY="uptime-gatus/config.yaml"
CONFIG_PATH="/config/config.yaml"
ETAG_PATH="${CONFIG_PATH}.etag"

remote_etag=$(aws s3api head-object \
  --bucket "${BUCKET}" \
  --key "${KEY}" \
  --query ETag --output text 2>/dev/null | tr -d '"')

if [[ -z "${remote_etag}" ]]; then
  echo "failed to fetch remote etag"
  exit 1
fi

local_etag=""
[[ -f "${ETAG_PATH}" ]] && local_etag=$(cat "${ETAG_PATH}")

if [[ "${remote_etag}" == "${local_etag}" ]]; then
  echo "config unchanged (etag=${remote_etag}), skipping download"
  exit 0
fi

echo "config changed (remote=${remote_etag} local=${local_etag:-<none>}), downloading"
aws s3 cp "s3://${BUCKET}/${KEY}" "${CONFIG_PATH}"
echo "${remote_etag}" > "${ETAG_PATH}"
echo "download success"
