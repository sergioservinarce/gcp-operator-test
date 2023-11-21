/*
Copyright 2023.

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

package controllers

import (
	"context"
	"fmt"
	"strings"

	errorgo "errors"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	runesv1alpha1 "github.com/sergioservinarce/gcp-operator-test/api/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RunesServiceReconciler reconciles a RunesService object
type RunesServiceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=runes.bancognb.com.py,resources=runesservices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=runes.bancognb.com.py,resources=runesservices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=runes.bancognb.com.py,resources=runesservices/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="networking.k8s.io",resources=ingresses,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the RunesService object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *RunesServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	runesService := &runesv1alpha1.RunesService{}

	//verificación del CRD
	log.Info("verifica que el CRD de tipo RunesService exista")
	err := r.Get(ctx, req.NamespacedName, runesService)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("RunesService CRD no encontrado")
			return ctrl.Result{}, nil
		}

		log.Error(err, "Error al obtener RunesService")
		return ctrl.Result{}, err
	}

	//Proceso Deployment
	log.Info("Verifica si el deployment ya existe, sino crea uno nuevo")
	existingDeployment := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: runesService.Name + "-deploy", Namespace: runesService.Namespace}, existingDeployment)
	if err != nil && errors.IsNotFound(err) {
		log.Info("al no encontrar, se define el nuevo deployment")
		newDeployment, err := r.getNewDeployment(runesService)
		if err != nil {
			log.Error(err, "Error al crear el Nuevo deployment")
			return ctrl.Result{}, err
		}
		err = r.Create(ctx, newDeployment)
		if err != nil {
			log.Error(err, "Error al crear el nuevo deployment", "Nampespace: ", newDeployment.Namespace, "Name: ", newDeployment.Name)
			return ctrl.Result{}, err
		}
		log.Info("Deployment creado correctamente")
		// Deployment creado correctamente - return and requeue
		return ctrl.Result{Requeue: true}, nil

	} else if err != nil {
		log.Error(err, "Error al obtener el deployment")
		return ctrl.Result{}, err
	}

	//Proceso Servicio Cluster IP
	log.Info("Inicio Proceso de creación de service ClusterIP")
	existingServiceClusterIP := &corev1.Service{}
	_ = existingServiceClusterIP

	newServiceClusterIP, err := r.getNewServiceClusterIP(runesService)
	if err != nil {
		return ctrl.Result{}, err
	}
	err = r.Get(context.TODO(), types.NamespacedName{Name: newServiceClusterIP.Name, Namespace: newServiceClusterIP.Namespace}, existingServiceClusterIP)
	if err != nil && errors.IsNotFound(err) {
		log.Info(fmt.Sprintf("Target service cluster %s doesn't exist, creating it", existingServiceClusterIP.Name))
		err = r.Create(context.TODO(), newServiceClusterIP)
		if err != nil {
			return ctrl.Result{}, err
		}
	} else {
		log.Info(fmt.Sprintf("Target service cluster %s exists, updating it now", existingServiceClusterIP))
		err = r.Update(context.TODO(), existingServiceClusterIP)
		if err != nil {
			return ctrl.Result{}, err
		}
	}
	//Proceso Servicio Ingress
	log.Info("Inicio Proceso de creación de service Ingress")
	existingIngress := &networkingv1.Ingress{}
	newIngress, err := r.getNewIngressService(runesService)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.Get(context.TODO(), types.NamespacedName{Name: newIngress.Name, Namespace: newIngress.Namespace}, existingIngress)

	if err != nil && errors.IsNotFound(err) {
		log.Info(fmt.Sprintf("Target IngressService  %s doesn't exist, creating it", newIngress.Name))
		err = r.Create(context.TODO(), newIngress)
		if err != nil {
			return ctrl.Result{}, err
		}
		log.Info("Ingress creado correctamente")
	} else {
		log.Info(fmt.Sprintf("Target IngressService %s exists, updating it now", existingIngress))
		err = r.Update(context.TODO(), existingIngress)
		if err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil

}

// SetupWithManager sets up the controller with the Manager.
func (r *RunesServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&runesv1alpha1.RunesService{}).
		Complete(r)
}

func (r *RunesServiceReconciler) getNewDeployment(runesService *runesv1alpha1.RunesService) (*appsv1.Deployment, error) {
	var replicaCount int32 = 1
	labelSelector, err := getLabelsForRunesService(runesService, "")
	if err != nil {
		return nil, err
	}
	labelSelectorPlatformLog, err := getLabelsForRunesService(runesService, "Apis")
	if err != nil {
		return nil, err
	}

	ImageName := getNameImage(runesService.Spec.Image.Repository)
	ImageRepositorySpec := runesService.Spec.Image.Repository

	ImageTagSpec := runesService.Spec.Image.Tag
	Image := ImageRepositorySpec + ":" + ImageTagSpec

	hostAliasSpec := getHostAliasSpec(runesService.Spec.HostAliases)
	stage := runesService.Spec.Apis.Stage
	ProfileValue := runesService.Spec.Main.Profile
	envSpecList := getEnvSpec(stage, runesService.Spec.ExtraEnv, ProfileValue)
	nameImagePullSecrets := "runes-services-" + runesService.Spec.Apis.Stage + "harbor-robot-secret"
	ImagePullSecretsD := []corev1.LocalObjectReference{
		{Name: nameImagePullSecrets},
	}
	_ = ImagePullSecretsD
	VolumeMountsD := []corev1.VolumeMount{
		{
			Name:      "config-server-key",
			MountPath: "/run/secrets/config-server.jks",
			SubPath:   "config-server.jks",
		},
	}

	_ = VolumeMountsD

	ContainersD := []corev1.Container{{
		Name:            ImageName,
		Image:           Image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Ports: []corev1.ContainerPort{{
			ContainerPort: 8080,
			Name:          "runes-services",
		}},
		Env: envSpecList,
		//VolumeMounts: VolumeMountsD,
	}}

	VolumesD := []corev1.Volume{
		{
			Name: "config-server-key",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: "runes-support-" + stage + "-config-server-secret",
					Items: []corev1.KeyToPath{
						{
							Key:  "config-server-" + stage + ".jks",
							Path: "config-server.jks",
						},
					},
				}},
		},
	}

	_ = VolumesD

	AffinityD := &corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{
					{
						MatchExpressions: []corev1.NodeSelectorRequirement{
							{
								Key:      "platform/stage",
								Operator: corev1.NodeSelectorOpIn,
								Values:   []string{stage},
							},
						},
					},
				},
			},
		},
	}

	_ = AffinityD

	//configuración de las secciones del Deployment

	podSpec := corev1.PodSpec{}

	//se arma el podSpec de acuerdo si tiene o no HostAlias
	if runesService.Spec.HostAliases != nil {
		podSpec = corev1.PodSpec{
			HostAliases: hostAliasSpec,
			//ImagePullSecrets: ImagePullSecretsD,
			Containers: ContainersD,
			//Volumes:          VolumesD,
			//Affinity: AffinityD,
		}
	} else {
		podSpec = corev1.PodSpec{
			//HostAliases:      hostAliasSpec,
			//ImagePullSecrets: ImagePullSecretsD,
			Containers: ContainersD,
			//Volumes:          VolumesD,
			//Affinity: AffinityD,
		}
	}

	SelectorSpec := &metav1.LabelSelector{
		MatchLabels: labelSelectorPlatformLog,
	}

	templateSpec := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: labelSelectorPlatformLog,
		},
		Spec: podSpec,
	}

	MetaDataDeploy := metav1.ObjectMeta{
		Name:      runesService.Name + "-deploy",
		Namespace: runesService.Namespace,
		Labels:    labelSelector,
	}

	SpecDeployment := appsv1.DeploymentSpec{
		Replicas: &replicaCount,
		Selector: SelectorSpec,
		Template: templateSpec,
	}

	newDeployment := &appsv1.Deployment{
		ObjectMeta: MetaDataDeploy,
		Spec:       SpecDeployment,
	}

	// Set runesService instance as the owner and controller
	ctrl.SetControllerReference(runesService, newDeployment, r.Scheme)

	return newDeployment, nil
}

func (r *RunesServiceReconciler) getNewServiceClusterIP(runesService *runesv1alpha1.RunesService) (*corev1.Service, error) {
	var port int32 = 80
	var targetPort int = 8080
	fullNameSvc := runesService.ObjectMeta.Name + "-svc"
	stageSpec := runesService.Spec.Apis.Stage
	name := "runes-services"
	fullNameApi := runesService.Name

	//nameRunesService := runesService.ObjectMeta.Name
	//if err != nil {
	//return nil, err
	//}
	selectorService := getSelectorForService(name, fullNameApi, stageSpec)

	labelSelector, err := getLabelsForRunesService(runesService, "")
	if err != nil {
		return nil, err
	}
	serv := &corev1.Service{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Service"},
		ObjectMeta: metav1.ObjectMeta{Name: fullNameSvc, Namespace: runesService.Namespace, Labels: labelSelector},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{{
				Port:       port,
				TargetPort: intstr.FromInt(targetPort),
			}},
			Selector: selectorService,
		},
	}
	return serv, nil
}

func (r *RunesServiceReconciler) getNewIngressService(runesService *runesv1alpha1.RunesService) (*networkingv1.Ingress, error) {
	//var pathTypeImplementationSpecific networkingv1.PathType = networkingv1.PathTypeImplementationSpecific
	//nameRunesService := nameIngress
	//stage := runesService.Spec.Apis.Stage
	//chartVersion := "1"
	className := "kong"

	laberlSelector, err := getLabelsForRunesService(runesService, "")
	if err != nil {
		return nil, err
	}
	pathsIngress, err := getPathIngressSpec(runesService)
	if err != nil {
		return nil, err
	}
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      runesService.Name + "-ingress",
			Namespace: runesService.Namespace,
			Labels:    laberlSelector,
			Annotations: map[string]string{
				//"konghq.com/plugins":             "runes-support-work-key-auth-kongplugin, runes-support-work-oidc-kongplugin",
				//"konghq.com/strip-path":       "true",
				//"kubernetes.io/ingress.class": "kong-" + stage + "-ic",
				"kubernetes.io/ingress.class": "kong", //kong
				//"plugins.konghq.com":          "runes-support-" + stage + "-key-auth-kongplugin, runes-support-" + stage + "-jwt-verify-kongplugin", //kong
				//"meta.helm.sh/release-name":      "apis-stage-accounts",
				//"meta.helm.sh/release-namespace": "apis-stage-ns",
			},
		},
		Spec: networkingv1.IngressSpec{
			//IngressClassName: pointer.String("kong-work-ic"),
			IngressClassName: &className, //pointer.String("kong"),
			Rules: []networkingv1.IngressRule{
				{IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: pathsIngress,
					},
				},
				},
			},
		},
	}

	return ingress, nil
}

func getLabelsForRunesService(runesService *runesv1alpha1.RunesService, platformLog string) (map[string]string, error) {
	labelMap := make(map[string]string)
	chartVersion := "1"
	name := "runes-services"
	stage := runesService.Spec.Apis.Stage
	//releaseName, err := getReleaseName(runesService.Name)
	//if err != nil {
	//	return nil, err
	//}

	fullNameApi := runesService.Name

	labelMap["app.kubernetes.io/name"] = name            //runesService.Name
	labelMap["app.kubernetes.io/instance"] = fullNameApi //releaseName
	if chartVersion != "" {
		labelMap["app.kubernetes.io/version"] = chartVersion
	}
	labelMap["app.kubernetes.io/managed-by"] = "Operator-sdk-go"
	labelMap["platform/stage"] = stage
	if platformLog != "" {
		labelMap["platform/log"] = platformLog
	}

	return labelMap, nil
}

func getReleaseName(runesServiceName string) (string, error) {

	var releaseName string

	indiceUltimoGuion := strings.LastIndex(runesServiceName, "-")
	if indiceUltimoGuion != -1 {
		releaseName = runesServiceName[:indiceUltimoGuion]
		return releaseName, nil
	} else {
		return "", errorgo.New("No se encontró un guion en la cadena.(getReleaseName)")
	}
}

func getHostAliasSpec(pHostAliases map[string]string) []corev1.HostAlias {
	HostAlias := []corev1.HostAlias{}

	for key, element := range pHostAliases {
		pHostname := []string{key}

		hAlias := corev1.HostAlias{IP: element,
			Hostnames: pHostname}
		HostAlias = append(HostAlias, hAlias)
	}

	return HostAlias
}

func getNameImage(pRepository string) string {
	parts := strings.Split(pRepository, "/")

	return parts[2]
}

func getEnvSpec(stage string, pEnvs map[string]string, pProfileVaule string) []corev1.EnvVar {

	EnvSpec := []corev1.EnvVar{{Name: "PROFILE", Value: pProfileVaule},
		{Name: "SERVER_PORT", Value: "8080"},
		{Name: "CONFIGSERVER_URI", Value: "http://runes-core-" + stage + "-config-server-svc.runes-" + stage + "-ns:9181"},
		{Name: "CONFIGSERVER_SERVICE", Value: "runes-core-" + stage + "-config-server-svc.runes-" + stage + "-ns"},
		{Name: "DISCOVERY_URI", Value: "http://runes-core-" + stage + "-discovery-peer1-svc.runes-" + stage + "-ns:8761/eureka/,http://runes-core-" + stage + "-discovery-peer2-svc.runes-" + stage + "-ns:8761/eureka/"},
		{Name: "DISCOVERY_PORT", Value: "8761"},
		{Name: "PEER1_SERVICE", Value: "runes-core-" + stage + "-discovery-peer1-svc.runes-" + stage + "-ns"}}

	for key, element := range pEnvs {
		env := corev1.EnvVar{Name: key, Value: element}
		EnvSpec = append(EnvSpec, env)
	}

	return EnvSpec
}

func getSelectorForService(name string, releaseName string, stage string) map[string]string {

	return map[string]string{"app.kubernetes.io/name": name,
		"app.kubernetes.io/instance": releaseName,
		"platform/stage":             stage}

}

func getPathIngressSpec(runesService *runesv1alpha1.RunesService) ([]networkingv1.HTTPIngressPath, error) {
	serviceName, error := getNameService(runesService.ObjectMeta.Name)
	if error != nil {
		return nil, error
	}

	pathIngress := getPathIngres(runesService.Spec.Runes.Stage) + "/" + serviceName

	var pathTypeImplementationSpecific networkingv1.PathType = networkingv1.PathTypeImplementationSpecific
	paths := []networkingv1.HTTPIngressPath{}

	paths = append(paths, networkingv1.HTTPIngressPath{Backend: networkingv1.IngressBackend{
		Service: &networkingv1.IngressServiceBackend{
			Name: runesService.Name + "-svc",
			Port: networkingv1.ServiceBackendPort{
				Number: 80,
			},
		},
	},
		Path:     pathIngress,
		PathType: &pathTypeImplementationSpecific,
	})

	pathsSpec := runesService.Spec.Ingress.Paths

	for _, pathValue := range pathsSpec {
		paths = append(paths, networkingv1.HTTPIngressPath{Backend: networkingv1.IngressBackend{
			Service: &networkingv1.IngressServiceBackend{
				Name: runesService.Name + "-svc",
				Port: networkingv1.ServiceBackendPort{
					Number: 80,
				},
			},
		},
			Path:     pathValue,
			PathType: &pathTypeImplementationSpecific,
		})
	}

	return paths, nil

}

func getPathIngres(stage string) string {
	var mainPath string
	if stage == "live" {
		mainPath = "/apis"
	} else {
		mainPath = "/apis/" + stage
	}

	return mainPath
}

func getNameService(runesServiceName string) (string, error) {

	var serviceName string

	indiceUltimoGuion := strings.LastIndex(runesServiceName, "-")
	if indiceUltimoGuion != -1 {
		serviceName = runesServiceName[indiceUltimoGuion+1:]
		return serviceName, nil
	} else {
		return "", errorgo.New("No se encontró un guion en la cadena.(getNameService)")
	}
}
