Proxies?? :/
How to scale? Message broker?

PSP
PNS:
 - Out only on 53/udp, 80,443,8080,4443,4343/tcp
 - Limit bandwidth (traffic shaping): https://kubernetes.io/docs/concepts/extend-kubernetes/compute-storage-net/network-plugins/#support-traffic-shaping
 - Limit connections (avoid RUDY attacks, can be done on a node level)

  iptables --new-chain RATE-LIMIT
  iptables --append OUTOPUT --match conntrack --ctstate NEW --jump RATE-LIMIT
  iptables --append RATE-LIMIT --match limit --limit 2/sec --limit-burst 5 --jump ACCEPT
  iptables --append RATE-LIMIT --jump REJECT # no dropping


set +m
for i in {0..10}; do { curl -o /dev/null -s -w "%{http_code}\n" 192.168.1.254 & } 2>/dev/null;  done
iptables --new-chain RATE-LIMIT
iptables --append RATE-LIMIT \
--match hashlimit \
--hashlimit-mode dstip \
--hashlimit-upto 2/sec \
--hashlimit-burst 3 \
--hashlimit-name conn_rate_limit \
--jump ACCEPT
iptables --append RATE-LIMIT --jump REJECT
iptables --append OUTPUT --match conntrack --ctstate NEW --jump RATE-LIMIT
for i in {0..10}; do { curl -o /dev/null -s -w "%{http_code}\n" 192.168.1.254 & } 2>/dev/null;  done
set -m
iptables -F
iptables --delete-chain RATE-LIMIT





https://strimzi.io/quickstarts/
