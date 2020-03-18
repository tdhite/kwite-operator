# Kwite Customer Resource Manifest
To create Kwites via
[Kwite-operator](https://github.com/tdhite/kwite-operator), the only
requirement is to fill out custom resource manifests describing the quite and
various scaling parameters. Details are [further below].

A manifest looks like the following example.

```yaml
apiVersion: web.kwite.site/v1beta1
kind: Kwite
metadata:
  name: kwite-1
spec:
  url: "/kwite"
  port: 8080
  image: "concourse.corp.local/kwite:latest"
  imagePullSecrets:
  - name: kwite-registry-creds
  targetcpu: 50
  minreplicas: 1
  maxreplicas: 10
  ready: "OK!"
  alive: "OK!"
  template: |
    This is a sample template that when executed x was {{ .x }}.

    The arccos of sin(Pi) is {{ Acos (Sin Pi) }} radians.
```

## Field Details 
This section details the values for the Kwite custom resource manifests.

* `apiVersion`:
At present this must be set to `web.kwite.site/v1beta1`

* `kind`:
This must be set to `Kwite`

* `metadata.name`:
This can be set to any name consistent with Kubernetes naming standards. The
name uniquely identifies the Kwite within a Kubernetes namespace. For example,
`kwite-1`. Note that the name of the Kwite is used by Kubernetes and the
Kwite-operator to identify it not just for management, but also to set DNS
naming cluster access to the Kwite.

* `spec.url`:
The URL to which the kwite will respond. For example, a url of `/kwite` would
cause the Kwite to respond to http://\<cluster-address\>/kwite.

* `spec.port`:
The internal (container) TCP port on which the Kwite will listen for incoming
HTTP connections. The default is `8080`.

* `spec.image`:
The container image identifier Kwite-operator should use for starting and
scaling Kwites. This should be set to the container registry path to the
container image and Kubernetes must have access to that registry in order to
pull the image. For example `concourse.corp.local/kwite:latest`.

* `spec.imagePullSecrets`:
An (optional) array of [Kubernetes registry
secrets](https://kubernetes.io/docs/concepts/containers/images/#specifying-imagepullsecrets-on-a-pod)
to use for image registries from which to pull Kwite and Kwite-operator
container images.

* `spec.memory`:
This sets the minimum amount of available memory necessary to schedule a Kwite
onto a Kubernetes node. The default is `64Mi`. For details on values, see
[Managing Compute Resources for
Containers](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/#meaning-of-cpu).

* `spec.cpu`:
This sets the minimum amount of available CPU necessary to schedule a Kwite
onto a Kubernetes node. The default is `200m`. For details on values, see
[Managing Compute Resources for
Containers](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/#meaning-of-cpu).

* `spec.targetcpu`:
The CPU target utilization per Kwite pod as specified by the [Horizontal Pod
Autoscaler](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/)
The default is `80`.

* `spec.minreplicas`:
The minimum number of Kwite pod instances that will exist at any time, to the
extent it is possible to start them.  The [Horizontal Pod
Autoscaler](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/)
handles scaling up and down relative to this value.

* `spec.maxreplicas`:
The maximum number of Kwite pod instances that will exist at any time.  The
[Horizontal Pod
Autoscaler](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/)
handles scaling up and down relative to this value.

* `spec.securityContext`:
Sets the security context for the Kwite containers. The default is:

```yaml
securityContext:
  allowPrivilegeEscalation: false
  readOnlyRootFilesystem: true
  runAsNonRoot: true
  runAsUser: 65534
```

For detailed information, see the [Kubernetes container security context
documentation](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/#set-the-security-context-for-a-container).

* `spec.ready`:
The [Go template](https://golang.org/pkg/text/template/) that the Kwite should
execute to determine if it is ready to begin servicing inbound HTTP requests.
For details, see Kubernetes
[probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/).

* `spec.alive`:
The [Go template](https://golang.org/pkg/text/template/) that the Kwite should
execute to determine if it is still alive. For details, see Kubernetes
[probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/).
See also the [Kwite
documentation](https://github.com/tdhite/kwite/blob/master/docs/kwites.md)
regarding its use of Go templating.

* `spec.template`:
The [Go template](https://golang.org/pkg/text/template/) that the Kwite should
execute as the response to HTTP requests on the Kwite.  See also the [Kwite
documentation](https://github.com/tdhite/kwite/blob/master/docs/kwites.md)
regarding its use of Go templating.
