#!/bin/bash
echo "+++++++++++++++++++++++++++++++++++++"
export $(cat /aws.env | xargs)
env > /aws.env
/usr/local/bin/aws s3 cp s3://simcloud-aws-props-dev/uptime-gatus/config.yaml /config/config.yaml
echo "download success"