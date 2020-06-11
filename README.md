GNSUPD -- calico Global Network Set UPdater Daemon.
===================================================

### Deprecation notice

This project is no longer maintained.

### Introduction

Calico has a concept of Global Network Sets(GNS). GNS is a 
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

### Description

GNSUPD is a daemon that converts `*.json` files in a
predefined directory into GNS resources. 
E.g., you have the following files in `/etc/ipsets/` directory:
```
first-set.json
second-set.json
```

Upon launch GNSUPD scans the directory, looking for files with `.json` suffixes.
Each file is supposed to contain `nets` JSON array of networks with CIDR masks: 
```
{ "nets": [ "192.168.10.0/24", "192.168.20.0/24", "8.8.8.8/32" ] }
``` 

For every json file found GNSUPD creates (or updates) a GNS resource. In our particular
example GNSUPD will create two GNS resources with names `first-set` and `second-set`, and assigns
them labels `first-set=true` and `second-set=true` respectively. If you have
configured the extra label, then GNSUPD will also add this extra label to every GNS resource with
the value `true`. 

After that GNSUPD will just sit there and wait for a HUP signal. When it receives 
a HUP signal, it rescans the directory, creates/updates GNS resources, and goes
back to sleep till the next HUP.

### Practical usage

GNSUPD is of no much use on its' own. The main idea behing GNSUPD is that
you have GNS resource files generated from templates and updated dynamically
via some other means. For example, you may want to generate GNS resources based
on information in [consul k/v store](https://consul.io) using [consul-template](https://github.com/hashicorp/consul-template).
Every time your GNS resource files get regenerated you should send a HUP signal
to the GNSUPD daemon to get the corresponding GNS resource updated.

### Configuration

GNSUPD is configured via environment variables:

* **GNSUPD_CONFIG_DIR**  
Path to the directory with set files. Defaults to **/etc/ipsets**.

* **GNSUPD_EXTRA_LABEL**  
The name of an additional label to be applied to every GNS resource created.

* **DATASTORE_TYPE**  
For talking with calico we need to know which datastore backend calico is using.
If you are not using calico with standalone etcd, then set this to `kubernetes`.

