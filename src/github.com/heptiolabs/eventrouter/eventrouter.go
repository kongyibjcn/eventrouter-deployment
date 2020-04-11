/*
Copyright 2017 Heptio Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/heptiolabs/eventrouter/sinks"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
	v1 "k8s.io/api/apps/v1"

	"github.com/streadway/amqp"
	V1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	appslisters "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
)

var (
	kubernetesWarningEventCounterVec = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "heptio_eventrouter_warnings_total",
		Help: "Total number of warning events in the kubernetes cluster",
	}, []string{
		"involved_object_kind",
		"involved_object_name",
		"involved_object_namespace",
		"reason",
		"source",
	})
	kubernetesNormalEventCounterVec = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "heptio_eventrouter_normal_total",
		Help: "Total number of normal events in the kubernetes cluster",
	}, []string{
		"involved_object_kind",
		"involved_object_name",
		"involved_object_namespace",
		"reason",
		"source",
	})
	kubernetesInfoEventCounterVec = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "heptio_eventrouter_info_total",
		Help: "Total number of info events in the kubernetes cluster",
	}, []string{
		"involved_object_kind",
		"involved_object_name",
		"involved_object_namespace",
		"reason",
		"source",
	})
	kubernetesUnknownEventCounterVec = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "heptio_eventrouter_unknown_total",
		Help: "Total number of events of unknown type in the kubernetes cluster",
	}, []string{
		"involved_object_kind",
		"involved_object_name",
		"involved_object_namespace",
		"reason",
		"source",
	})
)

// EventRouter is responsible for maintaining a stream of kubernetes
// system Events and pushing them to another channel for storage
type EventRouter struct {
	// kubeclient is the main kubernetes interface
	kubeClient kubernetes.Interface

	// store of events populated by the shared informer
	//eLister corelisters.EventLister
	eLister appslisters.DeploymentLister

	// returns true if the event store has been synced
	eListerSynched cache.InformerSynced

	// event sink
	// TODO: Determine if we want to support multiple sinks.
	eSink sinks.EventSinkInterface

	//Here we need to add a object used to store what object update event has been send
	updatingOjbect map[string]string

	//message queue connection
	mqChannel  *amqp.Channel
}



// NewEventRouter will create a new event router using the input params
// Define this new function to add deploymentInformer -- KongYi
func NewEventRouterTest (kubeClient kubernetes.Interface, deploymentInformer appsinformers.DeploymentInformer) *EventRouter {
	if viper.GetBool("enable-prometheus") {
		prometheus.MustRegister(kubernetesWarningEventCounterVec)
		prometheus.MustRegister(kubernetesNormalEventCounterVec)
		prometheus.MustRegister(kubernetesInfoEventCounterVec)
		prometheus.MustRegister(kubernetesUnknownEventCounterVec)
	}

	////here need to add code to init mq connection
	conn, err := amqp.Dial("amqp://admin:admin@10.122.46.24:5672/devops")

	if err != nil {
		glog.Infof("connect to mq failed: %v",err)
	}


	ch, err := conn.Channel()


	if err !=nil {
		glog.Infof("generate message queue channel failed: %v",err)
	}

	er := &EventRouter{
		kubeClient: kubeClient,
		eSink:      sinks.ManufactureSink(),
		updatingOjbect:  make(map[string]string),
		mqChannel: ch,
	}
	deploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    er.addEvent,
		UpdateFunc: er.updateEvent,
		DeleteFunc: er.deleteEvent,
	})
	er.eLister = deploymentInformer.Lister()
	er.eListerSynched = deploymentInformer.Informer().HasSynced
	return er
}




// NewEventRouter will create a new event router using the input params
//func NewEventRouter(kubeClient kubernetes.Interface, eventsInformer coreinformers.EventInformer) *EventRouter {
//	if viper.GetBool("enable-prometheus") {
//		prometheus.MustRegister(kubernetesWarningEventCounterVec)
//		prometheus.MustRegister(kubernetesNormalEventCounterVec)
//		prometheus.MustRegister(kubernetesInfoEventCounterVec)
//		prometheus.MustRegister(kubernetesUnknownEventCounterVec)
//	}
//
//	er := &EventRouter{
//		kubeClient: kubeClient,
//		eSink:      sinks.ManufactureSink(),
//	}
//	eventsInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
//		AddFunc:    er.addEvent,
//		UpdateFunc: er.updateEvent,
//		DeleteFunc: er.deleteEvent,
//	})
//	er.eLister = eventsInformer.Lister()
//	er.eListerSynched = eventsInformer.Informer().HasSynced
//	return er
//}

// Run starts the EventRouter/Controller.
	func (er *EventRouter) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer glog.Infof("Shutting down EventRouter")

	glog.Infof("Starting EventRouter")
	//fmt.Println("Starting EventRouter ....")

	// here is where we kick the caches into gear
	if !cache.WaitForCacheSync(stopCh, er.eListerSynched) {
		utilruntime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}
	//fmt.Println("Waiting Stop EventRouter Single")
	<-stopCh
}

// addEvent is called when an event is created, or during the initial list
//func (er *EventRouter) addEvent(obj interface{}) {
//	//e := obj.(*v1.Event)
//	e :=obj.(*v1.Deployment)
//	prometheusEvent(e)
//	er.eSink.UpdateEvents(e, nil)
//}

func (er *EventRouter) addEvent(obj interface{}) {
	//e := obj.(*v1.Event)
	//e:=obj.(string)
	//glog.Info(" add object %v",e)
	e :=obj.(*v1.Deployment)
	//prometheusEvent(e)
	//er.eSink.UpdateEvents(e, nil)
	glog.Infof("Event Add from the system:\n%v", e)
}


// updateEvent is called any time there is an update to an existing event
//func (er *EventRouter) updateEvent(objOld interface{}, objNew interface{}) {
//	eOld := objOld.(*v1.Event)
//	eNew := objNew.(*v1.Event)
//	prometheusEvent(eNew)
//	er.eSink.UpdateEvents(eNew, eOld)
//}
func (er *EventRouter) updateEvent(objOld interface{}, objNew interface{}) {
	//eOld := objOld.(*v1.Deployment)
	//eNew := objNew.(*v1.Deployment)
	//prometheusEvent(eNew)
	//er.eSink.UpdateEvents(eNew, eOld)
	//eOld:=objOld.(string)
	//eNew:=objNew.(string)
	//glog.Info(" update object from %v to %v",eOld,eNew)

	eNew := objNew.(*v1.Deployment)
	eOld := objOld.(*v1.Deployment)

	key_name := eNew.ObjectMeta.Name+"_"+eNew.ObjectMeta.Namespace
	//glog.Infof("deployment %v resourceVersion new reversion is: %v, old reversion is %v, current message is: %v", eNew.ObjectMeta.Name,eNew.ObjectMeta.ResourceVersion,eOld.ObjectMeta.ResourceVersion,eNew.Status.Conditions)
	if eNew.ObjectMeta.ResourceVersion != eOld.ObjectMeta.ResourceVersion || eNew.Status.Replicas != eNew.Status.AvailableReplicas {
		glog.Infof ("deployment %v is updating with image %v...( replica %v / availableReplicas %v / unavailableReplica %v )...",eNew.ObjectMeta.Name, eNew.Spec.Template.Spec.Containers[0].Image,eNew.Status.Replicas, eNew.Status.AvailableReplicas,eNew.Status.UnavailableReplicas)
		//Here continouse to send the message to mq then sync the status-- This find
		//但是会频繁的发送很多的消息，需要增加queue来限制发送的速率

		if _, exit:=er.updatingOjbect[key_name]; exit == false {

			er.updatingOjbect[key_name]="updating"
			glog.Infof ("add recored deployment %v update history. the whole list is %v", key_name, er.updatingOjbect)

			sync_message:="deployment "+ eNew.ObjectMeta.Name +" is updating with image "+ eNew.ObjectMeta.Name +"( replica "+ fmt.Sprint(eNew.Status.Replicas) +" / availableReplicas "+ fmt.Sprint(eNew.Status.AvailableReplicas) +" / unavailableReplica "+ fmt.Sprint(eNew.Status.UnavailableReplicas) +" )"

			glog.Infof ("send message %v to mq ", sync_message)

			err := er.mqChannel.Publish("deployment-status-sync", "mq-test", false, false, amqp.Publishing{
				ContentType: "text/plain",
				Body:        []byte(sync_message),
			})

			if err !=nil {
				glog.Infof ("send message to mq failed for deployment %v. The error is: %v ", key_name, err)
			}
		}
	}else{
		//glog.Infof ("deployment %v update complete with image %v...( replica %v / availableReplicas %v / unavailableReplica %v )...",eNew.ObjectMeta.Name, eNew.Spec.Template.Spec.Containers[0].Image,eNew.Status.Replicas, eNew.Status.AvailableReplicas,eNew.Status.UnavailableReplicas)
	    //send message to mq to update the
	    //
		if _, exit :=er.updatingOjbect[key_name]; exit == true {

			delete(er.updatingOjbect,key_name)
			glog.Infof ("delete recored deployment %v update history。 the whole list is %v", key_name, er.updatingOjbect)

			sync_message:="deployment "+ eNew.ObjectMeta.Name +" updated with image "+eNew.Spec.Template.Spec.Containers[0].Image

			err := er.mqChannel.Publish("deployment-status-sync", "mq-test", false, false, amqp.Publishing{
				ContentType: "text/plain",
				Body:        []byte(sync_message),
			})

			if err !=nil {
				glog.Infof ("send message to mq failed for deployment %v. The error is: %v ", key_name, err)
			}
		}
	}

	//if eNew.Status.Replicas != eNew.Status.AvailableReplicas {
	//	glog.Infof ("deployment %v is updating with image %v...( replica %v / availableReplicas %v / unavailableReplica %v )...",eNew.ObjectMeta.Name, eNew.Spec.Template.Spec.Containers[0].Image,eNew.Status.Replicas, eNew.Status.AvailableReplicas,eNew.Status.UnavailableReplicas)
	//}else{
	//	glog.Infof ("deployment %v update complete with image %v...( replica %v / availableReplicas %v / unavailableReplica %v )...",eNew.ObjectMeta.Name, eNew.Spec.Template.Spec.Containers[0].Image,eNew.Status.Replicas, eNew.Status.AvailableReplicas,eNew.Status.UnavailableReplicas)
	//}

	//if eJSONBytes, err := json.Marshal(eNew); err == nil {
	//	glog.Info("New Object:",string(eJSONBytes))
	//} else {
	//	glog.Warningf("Failed to json serialize event: %v", err)
	//}
	//if oldJSONBytes, err := json.Marshal(eOld); err == nil {
	//	glog.Info("Old Object:",string(oldJSONBytes))
	//} else {
	//	glog.Warningf("Failed to json serialize event: %v", err)
	//}


	//get deployment name
	//check deployment status
	// if "status":{"observedGeneration":2,"replicas":2,"updatedReplicas":1,"readyReplicas":1,"availableReplicas":1,"unavailableReplicas":1,
	// if old_resourceVersion != new_resourceVersion -- that means the deployment has been update
	//   if reason != "NewReplicaSetAvailable" that means deployment is still in updating

	//if eJSONBytes, err := json.Marshal(eNew); err == nil {
	//	glog.Info("Update Event:",string(eJSONBytes))
	//} else {
	//	glog.Warningf("Failed to json serialize event: %v", err)
	//}

	//glog.Info(" Update Evenet:",eNew)
}

func (er *EventRouter) updateEventNew(objOld interface{}, objNew interface{}) {
	//eOld := objOld.(*v1.Deployment)
	//eNew := objNew.(*v1.Deployment)
	////prometheusEvent(eNew)
	//er.eSink.UpdateEvents(eNew, eOld)
	////eOld:=objOld.(string)
	////eNew:=objNew.(string)
	////glog.Info(" update object from %v to %v",eOld,eNew)
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(objNew); err != nil {
		utilruntime.HandleError(err)
		return
	}

	//check if the scheduler exist, if yes, add to AddDelayDefined(item interface{})
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		glog.Errorf("invalid resource key: %s", key)
	}
	fmt.Println("&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&")
	fmt.Println(namespace)
	fmt.Println(name)
	fmt.Println("&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&")
}

// prometheusEvent is called when an event is added or updated
func prometheusEvent(event *V1.Event) {
	if !viper.GetBool("enable-prometheus") {
		return
	}
	var counter prometheus.Counter
	var err error

	switch event.Type {
	case "Normal":
		counter, err = kubernetesNormalEventCounterVec.GetMetricWithLabelValues(
			event.InvolvedObject.Kind,
			event.InvolvedObject.Name,
			event.InvolvedObject.Namespace,
			event.Reason,
			event.Source.Host,
		)
	case "Warning":
		counter, err = kubernetesWarningEventCounterVec.GetMetricWithLabelValues(
			event.InvolvedObject.Kind,
			event.InvolvedObject.Name,
			event.InvolvedObject.Namespace,
			event.Reason,
			event.Source.Host,
		)
	case "Info":
		counter, err = kubernetesInfoEventCounterVec.GetMetricWithLabelValues(
			event.InvolvedObject.Kind,
			event.InvolvedObject.Name,
			event.InvolvedObject.Namespace,
			event.Reason,
			event.Source.Host,
		)
	default:
		counter, err = kubernetesUnknownEventCounterVec.GetMetricWithLabelValues(
			event.InvolvedObject.Kind,
			event.InvolvedObject.Name,
			event.InvolvedObject.Namespace,
			event.Reason,
			event.Source.Host,
		)
	}

	if err != nil {
		// Not sure this is the right place to log this error?
		glog.Warning(err)
	} else {
		counter.Add(1)
	}
}

// deleteEvent should only occur when the system garbage collects events via TTL expiration
//func (er *EventRouter) deleteEvent(obj interface{}) {
//	e := obj.(*v1.Event)
//	// NOTE: This should *only* happen on TTL expiration there
//	// is no reason to push this to a sink
//	glog.V(5).Infof("Event Deleted from the system:\n%v", e)
//}

// deleteEvent should only occur when the system garbage collects events via TTL expiration
func (er *EventRouter) deleteEvent(obj interface{}) {
	e := obj.(*v1.Deployment)
	//e :=obj.(string)
	// NOTE: This should *only* happen on TTL expiration there
	// is no reason to push this to a sink

	if eJSONBytes, err := json.Marshal(e); err == nil {
		glog.Info("Deleted Event :",string(eJSONBytes))
	} else {
		glog.Warningf("Failed to json serialize event: %v", err)
	}
}
