#!/bin/bash

set -e -v
hotsub \
    --tasks ./bwa-alignment.csv \
    --script ./main.sh \
    --image otiai10/bwa \
    --aws-ec2-instance-type t2.large \
    --verbose
