# event-exporter

> 使用k8s的client-go实例化出sharedInformer, list and watch 集群的事件

默认k8s的事件只保留1小时, event-exporter可以收集集群出现的事件,方便后期查询

配合prometheus收集并展示k8s集群的警告事件

```
# HELP kubernetes_event_warning_total Total number of warning events in the kubernetes cluster
# TYPE kubernetes_event_warning_total counter
kubernetes_event_warning_total{involved_object_kind="Pod",involved_object_name="coredns-6dcc67dcbc-447zz",involved_object_namespace="kube-system",reason="Unhealthy",source_component="kubelet",source_host="dev-k8s-1"} 4
kubernetes_event_warning_total{involved_object_kind="Pod",involved_object_name="coredns-6dcc67dcbc-hws45",involved_object_namespace="kube-system",reason="Unhealthy",source_component="kubelet",source_host="dev-k8s-1"} 5
```

> 程序默认不暴露normal事件到metrics