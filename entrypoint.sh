#!/bin/sh

set -xe

/usr/local/bin/build-html --output /static --config /config.yml

chown -R app:app /static

su app -c "/bin/gostatic --files /static --addr :80 --compress-level 2"
