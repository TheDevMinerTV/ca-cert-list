#!/bin/sh

mkdir -p /static/certificates

CERT_HTML=$(
  cat <<EOF
<li>
  <a href="/certificates/{{FILENAME}}" download>
    <h2>{{NAME}}</h2>
    <span>{{DESCRIPTION}}</span>

    <div class="chips">
      <div class="chip">
        <span>
          <span>Expiry</span>
        </span>
        <span class="local-date-time" data-timestamp="{{EXPIRY_TIMESTAMP}}">{{EXPIRY}}</span>
      </div>

      <div class="chip">
        <span>
          <span>SHA256</span>
        </span>
        <span>
          <code>{{SHA256}}</code>
        </span>
      </div>

      <div class="chip">
        <span>
          <span>SHA1</span>
        </span>
        <span>
          <code>{{SHA1}}</code>
        </span>
      </div>

      <div class="chip">
        <span>
          <span>MD5</span>
        </span>
        <span>
          <code>{{MD5}}</code>
        </span>
      </div>
    </div>
  </a>
</li>
EOF
)

set -e

HTML=""
for CERT in /certs/*; do
  file="$CERT/cert.crt"
  certname=$(basename "$CERT")

  description=$(cat "$CERT/description.txt")
  expire_time=$(openssl x509 -enddate -noout -in "$file" | cut -d= -f2)
  expire_epoch=$(date -d "$expire_time" +%s)
  sha256=$(openssl x509 -in "$file" -noout -sha256 -fingerprint | cut -d= -f2)
  sha1=$(openssl x509 -in "$file" -noout -sha1 -fingerprint | cut -d= -f2)
  md5=$(openssl x509 -in "$file" -noout -md5 -fingerprint | cut -d= -f2)

  component=$(
    echo $CERT_HTML |
      sed "s|{{NAME}}|$certname|g" |
      sed "s|{{FILENAME}}|${certname}.crt|g" |
      sed "s|{{DESCRIPTION}}|$description|g" |
      sed "s|{{EXPIRY_TIMESTAMP}}|$expire_epoch|g" |
      sed "s|{{EXPIRY}}|$expire_time|g" |
      sed "s|{{SHA256}}|$sha256|g" |
      sed "s|{{SHA1}}|$sha1|g" |
      sed "s|{{MD5}}|$md5|g"
  )

  HTML="$HTML$component"

  cp "$file" "/static/certificates/$certname.crt"
done

# replace html
sed -i "s|{{HTML}}|$HTML|g" /static/index.html
