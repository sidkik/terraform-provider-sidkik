# terraform-provider-sidkik

This is a simple provider to help with firebase items that are not published via the google-beta provider. This provider is based on the google-beta provider.

It will handle creating new storage and firestore rules and the associated releases.

It also handles basic configuration of email auth and authorized domains. You must enable authentication first via the firebase console; otherwise, it will not find a config to update.

> Note: Unable to find a programatic way to enable the firebase auth or identity platform. The mobile sdk is not available except internally to google.

## Update Docs

## Local Dev

When using a local instance to try out the provider you can do the following:

Install locally

  mkdir -p ~/.terraform.d/plugins/registry.terraform.io/sidkik/sidkik/1.0.0/linux_amd64
  ln -s $GOPATH/bin/terraform-provider-sidkik ~/.terraform.d/plugins/registry.terraform.io/sidkik/sidkik/1.0.0/linux_amd64/terraform-provider-sidkik_v1.0.0

Clear lock and rebuild

  rm .terraform.lock.hcl && make && terraform init -upgrade


## Acc Tests

  make testacc TEST=./sidkik TESTARGS='-run=TestAccFirebaseAuthConfigDatasource_rule'