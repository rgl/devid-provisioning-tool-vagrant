# About

Vagrant test environment for running [devid-provisioning-tool](https://github.com/HewlettPackard/devid-provisioning-tool).

This is used as a playground before integrating it at https://github.com/rgl/spire-vagrant.

# Usage (Ubuntu 20.04)

Install [swtpm](https://github.com/stefanberger/swtpm) as described at https://github.com/rgl/swtpm-vagrant.

Install [Vagrant](https://github.com/hashicorp/vagrant), [vagrant-libvirt](https://github.com/vagrant-libvirt/vagrant-libvirt), and the [Ubuntu 20.04 base box](https://github.com/rgl/ubuntu-vagrant).

Start the Vagrant environment:

```bash
vagrant up --no-destroy-on-error --no-tty
```

Start the `provisioning-server`:

```bash
vagrant ssh
sudo -i
tpm-info
cd ~/devid-provisioning-tool
./bin/server/provisioning-server
```

In another shell, start the `provisioning-agent`:

```bash
vagrant ssh
sudo -i
cd ~/devid-provisioning-tool
./bin/agent/provisioning-agent
```
