/*
Copyright 2024.

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

package controller

import (
	"context"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog/v2"
	kmc "kmodules.xyz/client-go/client"
	readercomv1 "my.domain/kubebuilder/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// BookStoreReconciler reconciles a BookStore object
type BookStoreReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=reader.com.my.domain,resources=bookstores,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=reader.com.my.domain,resources=bookstores/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=reader.com.my.domain,resources=bookstores/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the BookStore object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func deploychanged(bk *readercomv1.BookStore, dep *appsv1.Deployment) bool {
	if *bk.Spec.Replicas != *dep.Spec.Replicas {
		*dep.Spec.Replicas = *bk.Spec.Replicas
		return true
	}
	if bk.Spec.Container.Image != dep.Spec.Template.Spec.Containers[0].Image {
		dep.Spec.Template.Spec.Containers[0].Image = bk.Spec.Container.Image
		return true
	}
	return false
}

func servicechanged(bk *readercomv1.BookStore, svc *corev1.Service) bool {
	if intstr.FromInt32(bk.Spec.Container.Port) != svc.Spec.Ports[0].TargetPort {
		svc.Spec.Ports[0].TargetPort = intstr.FromInt32(bk.Spec.Container.Port)
		return true
	}
	return false
}
func (r *BookStoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	fmt.Println("Reconsile called")
	fmt.Println("name: ", req.NamespacedName.Name, "\n namespace: ", req.NamespacedName.Namespace)

	var bk readercomv1.BookStore
	// getting bookstore resource
	if err := r.Get(ctx, req.NamespacedName, &bk); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	var bs = bk.DeepCopy()
	_, err := kmc.PatchStatus(ctx, r.Client, bs, func(obj client.Object) client.Object {
		in := obj.(*readercomv1.BookStore)
		if in.Status.State == "" {
			in.Status.State = "running"
		}
		return in
	})
	if err != nil {
		klog.Error("failed to update the book server")
		return ctrl.Result{}, err
	}

	fmt.Println("finding deployment")
	var dep appsv1.Deployment

	depName := types.NamespacedName{
		Name:      bk.Name,
		Namespace: bk.Namespace,
	}
	// getting deployment
	if err := r.Get(ctx, depName, &dep); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, err
		}
		dep = *newdeployment(&bk)
		if err := r.Create(ctx, &dep); err != nil {
			fmt.Println("unable to create deployment")
			return ctrl.Result{}, err
		}
		fmt.Println("deployment created")
	}
	// checking for deployment update
	if deploychanged(&bk, &dep) {
		fmt.Println("deployment changed")
		if err := r.Update(ctx, &dep); err != nil {
			return ctrl.Result{}, err
		}
		fmt.Println("deployment updated")
	}

	bs = bk.DeepCopy()
	_, err = kmc.PatchStatus(ctx, r.Client, bs, func(obj client.Object) client.Object {
		in := obj.(*readercomv1.BookStore)
		if in.Status.Dep == false {
			in.Status.Dep = true
		}
		return in
	})
	if err != nil {
		klog.Error("failed to update the book server")
		return ctrl.Result{}, err
	}
	// getting service
	fmt.Println("finding service")
	var svc corev1.Service
	svcName := types.NamespacedName{
		Name:      bk.Name,
		Namespace: bk.Namespace,
	}
	if err := r.Get(ctx, svcName, &svc); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, err
		}
		svc = *newservice(&bk)
		if err := r.Create(ctx, &svc); err != nil {
			fmt.Println("unable to create service")
			return ctrl.Result{}, err
		}
		fmt.Println("service created")
	}
	// checking for service update
	if servicechanged(&bk, &svc) {
		fmt.Println("service changed")
		if err := r.Update(ctx, &svc); err != nil {
			return ctrl.Result{}, err
		}
		fmt.Println("service updated")
	}
	bs = bk.DeepCopy()
	_, err = kmc.PatchStatus(ctx, r.Client, bs, func(obj client.Object) client.Object {
		in := obj.(*readercomv1.BookStore)
		in.Status.State = "ready"
		if in.Status.Svc == false {
			in.Status.Svc = true
		}
		return in
	})
	if err != nil {
		klog.Error("failed to update the book server")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

var (
	OwnerKey = ".metadata.controller"
	apiGVStr = readercomv1.GroupVersion.String()
	ourKind  = "BookStore"
)

// SetupWithManager sets up the controller with the Manager.
func (r *BookStoreReconciler) SetupWithManager(mgr ctrl.Manager) error {
	fmt.Println("In manager")
	// checking deployment owner
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &appsv1.Deployment{}, OwnerKey, func(object client.Object) []string {
		deployment := object.(*appsv1.Deployment)
		owner := metav1.GetControllerOf(deployment)
		if owner == nil {
			return nil
		}
		if owner.APIVersion != apiGVStr || owner.Kind != ourKind {
			return nil
		}
		return []string{owner.Name}
	}); err != nil {
		return err
	}
	// checking service owner
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Service{}, OwnerKey, func(object client.Object) []string {
		service := object.(*corev1.Service)
		owner := metav1.GetControllerOf(service)
		if owner == nil {
			return nil
		}
		if owner.APIVersion != apiGVStr || owner.Kind != ourKind {
			return nil
		}
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&readercomv1.BookStore{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

func newdeployment(bk *readercomv1.BookStore) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      bk.Name,
			Namespace: bk.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(bk, readercomv1.GroupVersion.WithKind(ourKind)),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "bookstore-app",
				},
			},
			Replicas: bk.Spec.Replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "bookstore-app",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "bookstore-app",
							Image: bk.Spec.Container.Image,
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									Protocol:      corev1.ProtocolTCP,
									ContainerPort: bk.Spec.Container.Port,
								},
							},
						},
					},
				},
			},
		},
	}
}
func newservice(bk *readercomv1.BookStore) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind: "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      bk.Name,
			Namespace: bk.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(bk, readercomv1.GroupVersion.WithKind(ourKind)),
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeNodePort,
			Selector: map[string]string{
				"app": "bookstore-app",
			},
			Ports: []corev1.ServicePort{
				{
					Port:       3200,
					TargetPort: intstr.FromInt32(bk.Spec.Container.Port),
					NodePort:   30002,
				},
			},
		},
	}
}
