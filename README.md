# sshbox

## What
Dummy ssh shell inside a container

## How
This connects to a local k8s cluster and deploys the container **using gvisor** (so yes, you need to use that)

## Quickstart!

```
####
# K3s
####
curl -sfL https://get.k3s.io | sh -

mkdir -p ~/.kube
ln -s /etc/rancher/k3s/k3s.yaml ~/.kube/config

####
# gvisor
####

# from gvisor site
(
  set -e
  ARCH=$(uname -m)
  URL=https://storage.googleapis.com/gvisor/releases/release/latest/${ARCH}
  wget ${URL}/runsc ${URL}/runsc.sha512 \
    ${URL}/containerd-shim-runsc-v1 ${URL}/containerd-shim-runsc-v1.sha512
  sha512sum -c runsc.sha512 \
    -c containerd-shim-runsc-v1.sha512
  rm -f *.sha512
  chmod a+rx runsc containerd-shim-runsc-v1
  sudo mv runsc containerd-shim-runsc-v1 /usr/local/bin
)

# backup 
cp /var/lib/rancher/k3s/agent/etc/containerd/config.toml{,.bak}
cp /var/lib/rancher/k3s/agent/etc/containerd/config.toml{,.tmpl}
cat <<EOF >> /var/lib/rancher/k3s/agent/etc/containerd/config.toml.tmpl

version = 2
[plugins."io.containerd.runtime.v1.linux"]
  shim_debug = true
[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
  runtime_type = "io.containerd.runc.v2"
[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runsc]
  runtime_type = "io.containerd.runsc.v1"
[plugins.cri.containerd.runtimes.runsc]
  runtime_type = "io.containerd.runsc.v1"
EOF

  

systemctl restart k3s

cat<<EOF | kubectl apply -f -
apiVersion: node.k8s.io/v1
kind: RuntimeClass
metadata:
  name: gvisor
handler: runsc
EOF

#Then use `runtimeClassName: gvisor` 
#
#cat << EOF | kubectl apply -f -
#apiVersion: v1
#kind: Pod
#metadata:
#  name: gvisor-test
#spec:
#  runtimeClassName: gvisor
#  containers:
#  - name: sshbox
#    image: pandry/ubuntubox
#    command: 
#      - sleep 
#      - infinity
#EOF
#
# kubectl exec -it gvisor-test -- zsh

####
# Go
####
yum install wget curl git tmux tar -y
wget https://dl.google.com/go/go1.18.linux-amd64.tar.gz
tar -zxvf go* -C /usr/local
rm -f go*
echo 'export GOROOT=/usr/local/go' | tee -a /etc/profile
echo 'export PATH=$PATH:/usr/local/go/bin' | tee -a /etc/profile
source /etc/profile
 
####
# SSH port
####
cp /etc/ssh/sshd_config{,.bak}
echo "Port 2222" >> /etc/ssh/sshd_config
systemctl reload sshd

####
# Anti RUDY DoS
####

iptables --new-chain RATE-LIMIT
iptables --append RATE-LIMIT \
--match hashlimit \
--hashlimit-mode dstip \
--hashlimit-upto 2/sec \
--hashlimit-burst 3 \
--hashlimit-name conn_rate_limit \
--jump ACCEPT
iptables --append RATE-LIMIT --jump REJECT
iptables --insert OUTPUT --match conntrack --ctstate NEW --jump RATE-LIMIT
iptables --insert KUBE-ROUTER-FORWARD --match conntrack --ctstate NEW --jump RATE-LIMIT
iptables --insert OUTPUT -p tcp -m multiport -m conntrack --ctstate NEW ! --dports 53,80,443,4443,8080 -j REJECT

####
# To run this software
####
git clone https://github.com/Pandry/sshbox
cd sshbox
go build .
tmux 
./sshbox


####
# Network policy to avoid SSH and stuff
####

cat << EOF | kubectl apply -f -
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: network-policy
  namespace: default
spec:
  podSelector:
    matchLabels:
      app: sshbox
  policyTypes:
  - Egress
  egress:
  - to:
    ports:
    - protocol: TCP
      port: 80
    - protocol: TCP
      port: 443
    - protocol: TCP
      port: 8080
    - protocol: TCP
      port: 8443
    - protocol: UDP
      port: 53
EOF
# Add ubuntu dnss

```