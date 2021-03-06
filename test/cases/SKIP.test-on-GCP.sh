#!/bin/bash

PROJROOT=$(dirname $(dirname $(cd $(dirname $0) && pwd)))

set -e -v

hotsub run \
    --tasks ${PROJROOT}/test/wordcount/wordcount.gcp.csv \
    --script ${PROJROOT}/test/wordcount/main.sh \
    --provider gcp \
    --google-zone asia-northeast1-a \
    --google-project genomondevel1 \
    --verbose
