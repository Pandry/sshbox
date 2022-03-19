package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"path/filepath"

	"github.com/gliderlabs/ssh"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	// github.com/spf13/viper
)

func main() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	deploymentsClient := clientset.AppsV1().Deployments(apiv1.NamespaceDefault)

	ssh.Handle(func(s ssh.Session) {
		//io.WriteString(s, "Hello world\n")
		defer s.Close()

		hash := sha1.New()
		pk := s.PublicKey().Marshal()
		_, err := hash.Write(pk)
		if err != nil {
			fmt.Println(err.Error())
		}
		userKeyFingerprint := hex.EncodeToString(hash.Sum(nil))
		io.WriteString(s, "Hi "+s.User()+"! This is your fingerprint: "+userKeyFingerprint+"\n") // Sarebbe bello aggiungere un punto al secondo finché il deployment non è completo

		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name: "demo-deployment",
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Ptr(1),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "sshbox",
					},
				},
				Template: apiv1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": "sshbox",
						},
					},
					Spec: apiv1.PodSpec{
						Containers: []apiv1.Container{
							{
								Name:  "sshbox-" + s.User() + userKeyFingerprint,
								Image: "alpine",
								TTY:   true,
							},
						},
					},
				},
			},
		}

		// Create Deployment
		io.WriteString(s, "Creating deployment...\n")
		_, err = deploymentsClient.Create(context.TODO(), deployment, metav1.CreateOptions{})
		if err != nil {
			panic(err)
		}
		io.WriteString(s, "Deployment created! Redirecting...\n")

		/*clientset.CoreV1().RESTClient().Post().
		Resource("pods").Name(result.pod).
		Namespace(result.Namespace).
		SubResource("exec")*/

		//defer deployment deletion
		defer func() {
			deploymentsClient.Delete(context.TODO(), deployment.GetObjectMeta().GetName(), metav1.DeleteOptions{})
			fmt.Println("Deleting ", deployment.GetObjectMeta().GetName())
		}()
	})

	publicKeyOption := ssh.PublicKeyAuth(func(ctx ssh.Context, key ssh.PublicKey) bool {
		return true // allow all keys, or use ssh.KeysEqual() to compare against known keys
	})

	log.Fatal(ssh.ListenAndServe(":2222", nil, publicKeyOption))

}

func int32Ptr(i int32) *int32 { return &i }
