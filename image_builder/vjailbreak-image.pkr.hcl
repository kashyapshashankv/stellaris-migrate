packer {
  required_plugins {
    qemu = {
      version = ">= 1.0.0"
      source  = "github.com/hashicorp/qemu"
    }
  }
}

source "qemu" "vjailbreak-image" {
  disk_image           = true
  skip_compaction      = true
  iso_url              = "vjailbreak-image.qcow2"
  iso_checksum         = "sha256:e0514d0ee287ca7fec7670e41ba67304f57eded5f4151f87734d7d3cc0a0d60a"
  iso_target_extension = "qcow2"
  output_directory     = "vjailbreak_qcow2"
  vm_name              = "vjailbreak-image.qcow2"
  disk_size            = "50G"
  format               = "qcow2"
  headless             = true
  accelerator          = "kvm"
  ssh_username         = "ubuntu"
  ssh_password         = "password"
  ssh_timeout          = "20m"
  cpus                 = 2
  memory               = 2048
  efi_boot             = false
  shutdown_command     = "echo 'password' | sudo -S shutdown -P now"
  boot_wait            = "10s"

  http_directory = "${path.root}/cloudinit/"

  qemuargs = [
    ["-smbios", "type=1,serial=ds=nocloud-net;s=http://{{ .HTTPIP }}:{{ .HTTPPort }}/"]
  ]
}

build {
  sources = ["source.qemu.vjailbreak-image"]

  provisioner "file" {
    source      = "${path.root}/deploy"
    destination = "/tmp/yamls"
  }

  provisioner "file" {
    source      = "${path.root}/scripts/install.sh"
    destination = "/tmp/install.sh"
  }

  provisioner "file" {
    source      = "${path.root}/configs/k3s.env"
    destination = "/tmp/k3s.env"
  }

  provisioner "file" {
    source      = "${path.root}/configs/daemonset.yaml"
    destination = "/tmp/daemonset.yaml"
  }

  provisioner "file" {
    source      = "${path.root}/configs/rsyncd.conf"
    destination = "/tmp/rsyncd.conf"
  }
  
  provisioner "file" {
    source      = "${path.root}/configs/env"
    destination = "/tmp/env"
  }

  provisioner "file" {
    source      = "${path.root}/configs/vjailbreak-settings.yaml"
    destination = "/tmp/vjailbreak-settings.yaml"
  }

  provisioner "file" {
    source      = "${path.root}/images"
    destination = "/home/ubuntu"
  }

  provisioner "file" {
    source      = "${path.root}/opensource.txt"
    destination = "/tmp/opensource.txt"
  }

  provisioner "shell" {
    inline = [
    "sudo mv /tmp/install.sh /etc/stellaris-migrate/install.sh",
    "sudo mv /tmp/k3s.env /etc/stellaris-migrate/k3s.env",
    "sudo mkdir -p image_builder/images",
    "sudo mv /home/ubuntu/images/* /etc/stellaris-migrate/images",
    "sudo mkdir -p /home/ubuntu/virtio-win",
    "sudo chown -R ubuntu:ubuntu /home/ubuntu/virtio-win",
    "sudo mv /etc/stellaris-migrate/images/virtio-win.iso /home/ubuntu/virtio-win/virtio-win.iso",
    "sudo mv /tmp/yamls /etc/stellaris-migrate/yamls",
    "sudo mv /tmp/rsyncd.conf /etc/stellaris-migrate/rsyncd.conf",
    "sudo mv /tmp/daemonset.yaml /etc/stellaris-migrate/yamls/daemonset.yaml",
    "sudo mv /tmp/env /etc/stellaris-migrate/env",
    "sudo mv /tmp/vjailbreak-settings.yaml /etc/stellaris-migrate/yamls/vjailbreak-settings.yaml",
    "sudo mv /tmp/opensource.txt /home/ubuntu/opensource.txt",
    "sudo chmod +x /etc/stellaris-migrate/install.sh",
    "sudo chown root:root /etc/stellaris-migrate/k3s.env",
    "sudo chmod 644 /etc/stellaris-migrate/k3s.env",
    "sudo chmod 644 /etc/stellaris-migrate/env",
    "sudo df -h",
    "echo '@reboot root /etc/stellaris-migrate/install.sh' | sudo tee -a /etc/crontab"
    ]
  }
}

