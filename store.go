/*
@Time : 2020/3/6 15:06
@Author : Tux
@Description :
*/

package main

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

var (
	// labelNames
	labelNames = []string{
		"involved_object_kind",
		"involved_object_name",
		"involved_object_namespace",
		"reason",
		"source_component",
		"source_host",
	}

	// Normal
	kubernetesNormalEventCounterVec = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "kubernetes_event_normal_total",
		Help: "Total number of normal events in the kubernetes cluster",
	}, labelNames)

	// Warning
	kubernetesWarningEventCounterVec = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "kubernetes_event_warning_total",
		Help: "Total number of warning events in the kubernetes cluster",
	}, labelNames)

	// Unknown
	kubernetesUnknownEventCounterVec = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "kubernetes_event_unknown_total",
		Help: "Total number of unknown events in the kubernetes cluster",
	}, labelNames)
)

type EventStore struct {
	// kubeClient is the main kubernetes interface
	kubeClient kubernetes.Interface

	// store of events populated by the shared informer
	eventLister corelisters.EventLister

	// returns true if the event store has been synced
	eventListerSynced cache.InformerSynced
}

func NewEventStore(kubeClient kubernetes.Interface, eventsInformer coreinformers.EventInformer) *EventStore {
	prometheus.MustRegister(kubernetesNormalEventCounterVec)
	prometheus.MustRegister(kubernetesWarningEventCounterVec)
	prometheus.MustRegister(kubernetesUnknownEventCounterVec)

	es := &EventStore{
		kubeClient: kubeClient,
	}
	eventsInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    es.addEvent,
		UpdateFunc: es.updateEvent,
		DeleteFunc: es.deleteEvent,
	})
	es.eventLister = eventsInformer.Lister()
	es.eventListerSynced = eventsInformer.Informer().HasSynced
	return es
}

// Run start the eventStore
func (es *EventStore) Run(stopCh <-chan struct{}) {
	// 处理 panic
	defer utilruntime.HandleCrash()
	defer glog.Infof("shutting down eventStore")
	glog.Info("starting eventStore")

	if !cache.WaitForCacheSync(stopCh, es.eventListerSynced) {
		utilruntime.HandleError(fmt.Errorf("time out waiting for caches to sync"))
		return
	}
	<-stopCh
}

// addEvent is called when an event is created, or during the initial list
func (es EventStore) addEvent(obj interface{}) {
	event := obj.(*corev1.Event)
	prometheusEvent(event)
	// glog.Infof("addEvent: %+v\n", event)
}

// updateEvent is called any time there is an update to an existing event
func (es EventStore) updateEvent(objOld interface{}, objNew interface{}) {
	// eventOld := objOld.(*corev1.Event)
	eventNew := objNew.(*corev1.Event)
	prometheusEvent(eventNew)
	// glog.Infof("updateEvent: eventOld: %+v \t eventNew: %+v\n", eventOld, eventNew)
}

// deleteEvent should only occur when the system garbage collects events via TTL expiration
func (es EventStore) deleteEvent(obj interface{}) {
	event := obj.(*corev1.Event)
	glog.V(5).Infof("deleteEvent: %v\n", event)
}

// prometheusEvent is called when an event is added or updated
func prometheusEvent(event *corev1.Event) {
	var counter prometheus.Counter
	var err error

	// Type of this event (Normal, Warning), new types could be added in the future
	switch event.Type {
	case "Normal":
		glog.V(3).Infof("Normal event: event:%+v\n", event)
		fmt.Println(*eventLevel)
		if *eventLevel == 1 {
			counter, err = kubernetesNormalEventCounterVec.GetMetricWithLabelValues(
				event.InvolvedObject.Kind,
				event.InvolvedObject.Name,
				event.InvolvedObject.Namespace,
				event.Reason,
				event.Source.Component,
				event.Source.Host,
			)
			if err != nil {
				glog.Warning(err)
			} else {
				counter.Add(1)
			}
		}

	case "Warning":
		glog.Infof("Warning event: event:%+v\n", event)
		counter, err = kubernetesWarningEventCounterVec.GetMetricWithLabelValues(
			event.InvolvedObject.Kind,
			event.InvolvedObject.Name,
			event.InvolvedObject.Namespace,
			event.Reason,
			event.Source.Component,
			event.Source.Host,
		)
		if err != nil {
			glog.Warning(err)
		} else {
			counter.Add(1)
		}
	default:
		glog.Infof("Unknown event: event:%+v\n", event)
		counter, err = kubernetesUnknownEventCounterVec.GetMetricWithLabelValues(
			event.InvolvedObject.Kind,
			event.InvolvedObject.Name,
			event.InvolvedObject.Namespace,
			event.Reason,
			event.Source.Component,
			event.Source.Host,
		)
		if err != nil {
			glog.Warning(err)
		} else {
			counter.Add(1)
		}
	}

}
