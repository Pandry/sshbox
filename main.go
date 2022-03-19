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
	"time"

	"github.com/gliderlabs/ssh"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
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

	podsClient := clientset.CoreV1().Pods(apiv1.NamespaceDefault)

	ssh.Handle(func(s ssh.Session) {
		//io.WriteString(s, "Hello world\n")
		defer s.Close()

		hash := sha1.New()
		pk := s.PublicKey().Marshal()
		_, err := hash.Write(pk)
		if err != nil {
			fmt.Println(err.Error())
		}
		// check if pods alredy exists
		// 		check it's not in terminating state/ is in running state
		userKeyFingerprint := hex.EncodeToString(hash.Sum(nil))
		io.WriteString(s, "Hi "+s.User()+"! This is your fingerprint: "+userKeyFingerprint+"\n") // Sarebbe bello aggiungere un punto al secondo finché il deployment non è completo
		podName := "sshbox-" + s.User() + "-" + userKeyFingerprint

		pod := &apiv1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:   podName,
				Labels: map[string]string{"app": "sshbox"},
			},
			Spec: apiv1.PodSpec{
				Containers: []apiv1.Container{
					{
						Name:  "sshbox",
						Image: "alpine",
						TTY:   true,
					},
				},
			},
		}

		// Create Deployment
		io.WriteString(s, "Creating deployment...\n")
		result, err := podsClient.Create(context.TODO(), pod, metav1.CreateOptions{})
		if err != nil {
			panic(err)
		}
		time.Sleep(2 * time.Second)
		io.WriteString(s, "Deployment created! Redirecting...\n")

		req := clientset.CoreV1().RESTClient().Post().Resource("pods").Name(result.Name).Namespace(result.Namespace).SubResource("exec")
		req.VersionedParams(&apiv1.PodExecOptions{
			TypeMeta:  metav1.TypeMeta{},
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
			Container: "sshbox",
			Command:   []string{"sh"},
		}, scheme.ParameterCodec)

		exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
		if err != nil {
			panic(err)
		}

		err = exec.Stream(remotecommand.StreamOptions{
			Stdin:  s,
			Stdout: s,
			Stderr: s,
		})

		if err != nil {
			io.WriteString(s, err.Error()+"\n")
		}

		io.WriteString(s, "Quitted!\n")

		//defer deployment deletion
		defer func() {
			podsClient.Delete(context.TODO(), pod.GetObjectMeta().GetName(), metav1.DeleteOptions{})
			fmt.Println("Deleting ", pod.GetObjectMeta().GetName())
		}()
	})

	publicKeyOption := ssh.PublicKeyAuth(func(ctx ssh.Context, key ssh.PublicKey) bool {
		return true // allow all keys, or use ssh.KeysEqual() to compare against known keys
	})

	log.Fatal(ssh.ListenAndServe(":2222", nil, publicKeyOption))

}
