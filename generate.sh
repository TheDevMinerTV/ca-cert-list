#!/bin/ash

mkdir -p /static/certificates

HTML=""

set -xe

for certname; do
  sha256=$(openssl x509 -in "$certfile" -noout -sha256 -fingerprint | cut -d= -f2)
  sha1=$(openssl x509 -in "$certfile" -noout -sha1 -fingerprint | cut -d= -f2)
  md5=$(openssl x509 -in "$certfile" -noout -md5 -fingerprint | cut -d= -f2)

  set +x
  component=$(
    echo $CERT_HTML |
      sed "s#{{NAME}}#$certname#g" |
      sed "s#{{FILENAME}}#${certname}.crt#g" |
      sed "s#{{DESCRIPTION}}#$description#g" |
      sed "s#{{EXPIRY_TIMESTAMP}}#$expire_epoch#g" |
      sed "s#{{EXPIRY}}#$expire_time#g" |
      sed "s#{{SHA256}}#$sha256#g" |
      sed "s#{{SHA1}}#$sha1#g" |
      sed "s#{{MD5}}#$md5#g"
  )

  HTML="$HTML$component"
  set -x

  cp "$certfile" "/static/certificates/$certname.crt"
done
