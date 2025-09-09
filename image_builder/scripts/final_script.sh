#!/bin/bash

set -x


sudo mkdir -p /etc/stellaris/images
sudo mkdir -p /etc/stellaris/k3s-setup/
sudo mv /home/ubuntu/scripts/install.sh /etc/stellaris/install.sh
sudo mv /home/ubuntu/configs/k3s.env /etc/stellaris/k3s.env
sudo mv /home/ubuntu/images/* /etc/stellaris/images
sudo mkdir -p /home/ubuntu/virtio-win
sudo chown -R ubuntu:ubuntu /home/ubuntu/virtio-win
sudo mkdir -p /var/lib/rancher/k3s/agent/images/
sudo mv /home/ubuntu/deploy /etc/stellaris/yamls
sudo mv /home/ubuntu/ingress-nginx /etc/stellaris/ingress-nginx
sudo mv /home/ubuntu/configs/rsyncd.conf /etc/stellaris/rsyncd.conf
sudo mv /home/ubuntu/configs/daemonset.yaml /etc/stellaris/yamls/daemonset.yaml
sudo mv /home/ubuntu/configs/env /etc/stellaris/env
sudo mv /home/ubuntu/configs/stellaris-migrate-settings.yaml /etc/stellaris/yamls/stellaris-migrate-settings.yaml
sudo chmod +x /etc/stellaris/install.sh
sudo chown root:root /etc/stellaris/k3s.env
sudo chmod 644 /etc/stellaris/k3s.env
sudo chmod 644 /etc/stellaris/env
virtiowin="https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/stable-virtio/virtio-win.iso"
# Download virtio-win.iso
echo "[*] Downloading virtio-win.iso"
sudo wget -O /etc/stellaris/images/virtio-win.iso "$virtiowin"
sudo mv /etc/stellaris/images/virtio-win.iso /home/ubuntu/virtio-win/virtio-win.iso
# install k3s binary and tar file it needs. 
echo "[*] Downloading k3s-install.sh"
sudo curl -sfL https://get.k3s.io -o /etc/stellaris/k3s-setup/k3s-install.sh
sudo chmod +x /etc/stellaris/k3s-setup/k3s-install.sh

echo "[*] Downloading k3s binary"
sudo curl -L https://github.com/k3s-io/k3s/releases/download/v1.33.1%2Bk3s1/k3s -o /usr/local/bin/k3s
sudo chmod +x /usr/local/bin/k3s

echo "[*] Downloading k3s-airgap-images-amd64.tar.zst"
sudo curl -LO https://github.com/k3s-io/k3s/releases/download/v1.33.1%2Bk3s1/k3s-airgap-images-amd64.tar.zst
echo "[*] Moving k3s-airgap-images-amd64.tar.zst to /var/lib/rancher/k3s/agent/images/"
sudo mv k3s-airgap-images-amd64.tar.zst /var/lib/rancher/k3s/agent/images/

echo "[*] Downloading Helm binary"
sudo curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

sudo apt update -y
sudo apt install containerd -y



echo "[*] Fetching latest ingress-nginx controller tag from GitHub..."
controller_tag=$(curl -s https://api.github.com/repos/kubernetes/ingress-nginx/releases/latest | jq -r .tag_name)
controller_tag="controller-v1.12.2"
echo "[+] Latest tag: $controller_tag"

# Download values.yaml from GitHub to extract image info
values_url="https://raw.githubusercontent.com/kubernetes/ingress-nginx/${controller_tag}/charts/ingress-nginx/values.yaml"
values_file=$(mktemp)
curl -sL "$values_url" -o "$values_file"

# Extract digests
controller_digest=$(awk '/controller:/{f=1} f && /digest:/{print $2; exit}' "$values_file")
certgen_digest=$(awk '/kube-webhook-certgen/{f=1} f && /digest:/{print $2; exit}' "$values_file")

# pull image references for nginx
controller_image="registry.k8s.io/ingress-nginx/controller@${controller_digest}"
certgen_image="registry.k8s.io/ingress-nginx/kube-webhook-certgen@${certgen_digest}"

kube_state_metrics="registry.k8s.io/kube-state-metrics/kube-state-metrics:v2.13.0"
prometheus_adapter="registry.k8s.io/prometheus-adapter/prometheus-adapter:v0.12.0"
prometheus="quay.io/prometheus/prometheus:v2.54.1"
alertmanager="quay.io/prometheus/alertmanager:v0.27.0"
blackbox_exporter="quay.io/prometheus/blackbox-exporter:v0.25.0"
node_exporter="quay.io/prometheus/node-exporter:v1.8.2"
pushgateway="quay.io/prometheus/pushgateway:v1.5.0"
kube_rbac_proxy="quay.io/brancz/kube-rbac-proxy:v0.19.1"
prometheus_config_reloader="quay.io/prometheus-operator/prometheus-config-reloader:v0.76.0"
prometheus_operator="quay.io/prometheus-operator/prometheus-operator:v0.76.0"
configmap_reload="ghcr.io/jimmidyson/configmap-reload:v0.13.1"
grafana="docker.io/grafana/grafana:11.2.0"
alpine="quay.io/platform9/vjailbreak:alpine"


v2v_helper="quay.io/stellaris/stellaris-migrate-v2v-helper:0.0.1"
controller="quay.io/stellaris/stellaris-migrate-controller:0.0.1"
ui="quay.io/stellaris/stellaris-migrate-ui:0.0.1"
vpwned="quay.io/stellaris/stellaris-migrate-vpwned:0.0.1"
# Download and export images
images=(
  "$controller_image"
  "$certgen_image"
  "$kube_state_metrics"
  "$prometheus_adapter"
  "$prometheus"
  "$alertmanager"
  "$blackbox_exporter"
  "$node_exporter"
  "$pushgateway"
  "$prometheus_config_reloader"
  "$prometheus_operator"
  "$configmap_reload"
  "$grafana"
  "$alpine"
  "$v2v_helper"
  "$controller"
  "$ui"
  "$vpwned"
)



for img in "${images[@]}"; do
  echo "[*] Pulling $img"
  sudo ctr i pull "$img"

  tag=$(echo "$img" | cut -d'@' -f1)
  fname=$(echo "$tag" | tr '/:@' '_')

  echo "[*] Exporting to $fname.tar"
  sudo ctr i export "/etc/stellaris/images/$fname.tar" "$img"
done

echo "[âœ”] All images downloaded and exported as tar files."
