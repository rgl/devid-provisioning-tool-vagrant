#!/bin/bash
source /vagrant/lib.sh

devid_provisioning_version="${1:-b912ef2c19571093dfacd0a6721dd1e6f6299768}"; shift || true
cfssl_version="${1:-1.6.1}"; shift || true

# echo the executed commands to stderr.
set -x

# ensure the host share directory exists.
if [ ! -d /vagrant/share ]; then
    install -d /vagrant/share
fi

# download and install cfssl.
for name in cfssl cfssljson; do
    artifact_url="https://github.com/cloudflare/cfssl/releases/download/v$cfssl_version/${name}_${cfssl_version}_linux_amd64"
    artifact_path="/vagrant/share/$(basename "$artifact_url")"
    if [ ! -f "$artifact_path" ]; then
        wget -qO "$artifact_path" "$artifact_url"
    fi
    install -m 755 $artifact_path /usr/local/bin/$name
done

# create the ca and certificates.
mkdir -p ~/devid-provisioning-ca
pushd ~/devid-provisioning-ca
cfssl gencert -initca /vagrant/devid-provisioning-ca-csr.json \
    | cfssljson -bare ca -
cfssl gencert \
    -ca ca.pem \
    -ca-key ca-key.pem \
    /vagrant/devid-provisioning-server-csr.json \
    | cfssljson -bare server
popd


# get the source code.
if [ ! -d ~/devid-provisioning-tool ]; then
    git clone https://github.com/HewlettPackard/devid-provisioning-tool.git ~/devid-provisioning-tool
fi

# checkout the required version.
cd ~/devid-provisioning-tool
git checkout $devid_provisioning_version

# build.
make build

# install the keys and certificates.
install -m 644 ~/devid-provisioning-ca/ca.pem ~/devid-provisioning-tool/conf/agent/server-ca.crt
install -m 644 ~/devid-provisioning-ca/ca.pem ~/devid-provisioning-tool/conf/server/provisioning-ca.crt
openssl pkcs8 -topk8 -nocrypt -in ~/devid-provisioning-ca/ca-key.pem -out ~/devid-provisioning-tool/conf/server/provisioning-ca.key
install -m 644 ~/devid-provisioning-ca/server.pem ~/devid-provisioning-tool/conf/server/server.crt
install -m 640 ~/devid-provisioning-ca/server-key.pem ~/devid-provisioning-tool/conf/server/server.key
install -m 644 /vagrant/share/swtpm-localca-rootca-crt.pem ~/devid-provisioning-tool/conf/server/manufacturer-ca.crt
cat /vagrant/share/swtpm-localca-crt.pem >>~/devid-provisioning-tool/conf/server/manufacturer-ca.crt
