Vagrant.configure('2') do |config|
  config.vm.box = 'ubuntu-20.04-amd64'

  config.vm.provider :libvirt do |lv, config|
    lv.cpus = 4
    lv.cpu_mode = 'host-passthrough'
    #lv.nested = true
    lv.memory = 1*1024
    lv.keymap = 'pt'
    lv.tpm_type = 'emulator'
    lv.tpm_model = 'tpm-crb'
    lv.tpm_version = '2.0'
    config.vm.synced_folder '.', '/vagrant', type: 'nfs', nfs_version: '4.2', nfs_udp: false
  end

  config.vm.provision :shell, path: 'provision-base.sh'
  config.vm.provision :shell, path: 'provision-go.sh'
  config.vm.provision :shell, path: 'provision-tpm-info.sh'
  config.vm.provision :shell, path: 'provision-devid-provisioning-tool.sh'

  config.trigger.before :up do |trigger|
    trigger.run = {
      inline: '''bash -euc \'
mkdir -p share
artifacts=(
  "/var/lib/swtpm-localca/swtpm-localca-rootca-cert.pem swtpm-localca-rootca-crt.pem"
  "/var/lib/swtpm-localca/issuercert.pem swtpm-localca-crt.pem"
)
for artifact in "${artifacts[@]}"; do
  echo "$artifact" | while read artifact path; do
    cp "$artifact" "share/$path"
  done
done
\'
'''
    }
  end
end
