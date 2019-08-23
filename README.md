# Coastie-operator

The Coastie [Operator][1] monitors the health of your kubernetes cluster by running a series of tests that can be turned on or off in the Coastie CustomResource.

[Coastie][4] is short for members of the Coast Guard, who among other things monitor coastal waters, kubernetes is nautically themed.

## Getting Started

See the [tutorial][2] which provides a quick-start guide for users of the Coastie Operator.

## Features

The Coastie Operator monitors the following resources if udp, tcp, and http tests are enabled:

- K8s HTTP Ingress
  - DNS
  - TCP connection
  - Router
  - k8s Service
  - k8s DaemonSet
  - Pod startup latency
  - Slack notification if deadline exceeded
  - Image pull
- TCP
  - TCP connection
  - k8s Service
  - k8s DaemonSet
  - Pod startup latency
  - Slack notification if deadline exceeded
  - Image pull
- UDP
  - UDP connection
  - k8s Service
  - k8s DaemonSet
  - Pod startup latency
  - Slack notification if deadline exceeded
  - Image pull

## Tested against

 * Openshift 3.11
 * Though it should work against k8s cluster

## Contributing

Feel Free to open issues and send pull requests. We welcome them.

See [testing][3] for instructions on how to build and test the operator

## License
GPLv2, good enough for Linus, good enough for us.

See [LICENSE](LICENSE) for more details.

[1]: https://coreos.com/blog/introducing-operators.html
[2]: docs/tutorial.md
[3]: docs/testing.md
[4]: https://coastie.soh.re
