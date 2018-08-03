GNSUPD -- calico global network set updater.
============================================

*** Description

Calico has a concept of Global Network Sets. GNS is a 
a list of CIDR networks, sharing the same labels. Once
you have created a GNS, you can use it in network policies. 
For example:

Define a GNS:
```
apiVersion: projectcalico.org/v3
kind: GlobalNetworkSet
metadata:
  name: trusted-networks
  labels:
    trusted-networks: "true"
spec:
  nets:
  - 172.16.0.0/12
  - 127.0.0.0/8
  - 10.0.0.0/8
  - 192.168.0.0/16
```

Define host endpoints:
```
---
apiVersion: projectcalico.org/v3
kind: HostEndpoint
metadata:
  name: ny1
  labels:
    fire: walled
spec:
  interfaceName: eth0
  node: k8s-email-ny-stage-2
---
apiVersion: projectcalico.org/v3
kind: HostEndpoint
metadata:
  name: am1
  labels:
    fire: walled
spec:
  interfaceName: eth0
  node: k8s-email-am-stage-2
```

Finally, use it all in a policy:
```
apiVersion: projectcalico.org/v3
kind: GlobalNetworkPolicy
metadata:
  name: allow-all-from-trusted-networks
spec:
  selector: fire == 'walled'
  types:
  - Ingress
  - Egress
  ingress:
  - action: Allow
    protocol: TCP
    source:
      selector: trusted-networks == "true"
  - action: Allow
    protocol: UDP
    source:
      selector: trusted-networks == "true"
  egress:
  - action: Allow
```

*** Configuration

GNSUPD is configured via environment variables:

* **GNSUPD_NETWORKS_FILE**  
Path to the file with networks for the set. Must be JSON of the following format: 
```
{ "nets": [ "10.0.0.0/8", "192.168.10.0/23" ] }
```

* **GNUSPD_SET_NAME**  
A name of the GNS. GNSUPD will also assign the label "GNSUPD_SET_NAME: true" to the 
created set.

* **DATASTORE_TYPE**  
For talking with calico with need to know which datastore backend calico is using.
If you are not using calico with standalone etcd, then set this to `kubernetes`.

