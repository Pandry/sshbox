package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	sshp "golang.org/x/crypto/ssh"

	"github.com/gliderlabs/ssh"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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
	podNamespace := apiv1.NamespaceDefault
	podsClient := clientset.CoreV1().Pods(podNamespace)

	ssh.Handle(func(s ssh.Session) {
		//io.WriteString(s, "Hello world\n")
		defer s.Close()

		if !(s.User()[:6] == "unipi-") {
			io.WriteString(s, `Pandry says I shouldn't talk to strangers, but here's a nice pizza recipie:
Flour 200g (possibly with 14g of proteins per 100g)
Water 300ml 
Salt  10g 
Yeast 4g 
Extravirgin Oil 80g (NOT ur nut!!)

Mix all togheder (salt ad the end!)
Let it rest for 20 mins; then fold it every 30 or so minutes per 3 times, then wait a couple of hours; Do little 300g dough balls and fold them; roll the dough after ~2 hours and let it rest for 1 hour; Bake @ max @ ~ 20 mins
Enjoy!

And here a good song list: 
https://www.youtube.com/watch?v=55OJ17cHeJA
https://www.youtube.com/watch?v=-UkSdSlY1YA
https://www.youtube.com/watch?v=xHE5g9YgkFg
https://www.youtube.com/watch?v=vBZ5SLJmfdw
https://www.youtube.com/watch?v=OO4ib5xC7Fk
https://www.youtube.com/watch?v=1uB2ZRjbtbY

Have a good day! :)
`)
			return
		}

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
		fmt.Println("Connesso utente '", s.User(), "' dall'IP ", s.RemoteAddr())
		podName := "sshbox-" + s.User() + "-" + userKeyFingerprint

		//podList, err := podsClient.List(context.TODO(), metav1.ListOptions{})

		gotPod, err := podsClient.Get(context.TODO(), podName, metav1.GetOptions{})
		podFound := true
		if err != nil {
			errStr := err.Error()
			//this is horrible
			if errStr[len(errStr)-10:] == " not found" {
				podFound = false
			} else {
				io.WriteString(s, fmt.Sprint(err, gotPod, "Error occurred while searching for the pod: "+err.Error()))
				return
			}
		}
		//io.WriteString(s, string(gotPod.Status.Phase)+" - "+gotPod.Status.String()+" - ")

		//Container may be terminating but it would drop in a shell anyway :c
		trueBool := true
		if !podFound {
			err = nil
			pod := &apiv1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:   podName,
					Labels: map[string]string{"app": "sshbox"},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:  "sshbox",
							Image: "pandry/ubuntubox",
							//TTY:   true,
							Resources: apiv1.ResourceRequirements{
								Limits: apiv1.ResourceList{
									apiv1.ResourceCPU: *resource.NewQuantity(1, resource.BinarySI),
									"memory":          *resource.NewQuantity(200000000, resource.BinarySI),
								},
							},
							Command: []string{"sleep", "infinity"},
							//Args: []string{"zsh"},
							//Args: []string{"zsh"},
						},
					},
					Hostname:          "sshbox",
					SetHostnameAsFQDN: &trueBool,
				},
			}

			// Create Deployment
			io.WriteString(s, "Creating pod..")
			_, err := podsClient.Create(context.TODO(), pod, metav1.CreateOptions{})
			if err != nil {
				io.WriteString(s, "Error occurred during pod creation: "+err.Error())
				fmt.Print(pod)
				return
			}
			//defer deployment deletion
			defer func() {
				var i int64 = 0
				podsClient.Delete(context.TODO(), podName, metav1.DeleteOptions{GracePeriodSeconds: &i})
				fmt.Println("Deleting ", podName)
			}()

			// Remove the wait time
			podReady := false
			for !podReady {
				gotPod, err := podsClient.Get(context.TODO(), podName, metav1.GetOptions{})
				if err != nil {
					errStr := err.Error()
					switch {
					case errStr[len(errStr)-10:] == " not found":
						io.WriteString(s, "\nSi è verificato un errore, il pod non è stato creato! contatta pandry!")
					default:
						io.WriteString(s, "\nSi è verificato un errore: "+err.Error())
					}
				}
				switch gotPod.Status.Phase {
				case "Pending":
					io.WriteString(s, ".")
				case "Running":
					io.WriteString(s, "\nYay! Servert creato! Inizializzo il wormhole...\nFatto!\nBenvenuto su Linux!\n")
					podReady = true
				default:
					io.WriteString(s, "Uhm, scrivi questo a Pandry: "+string(gotPod.Status.Phase))
				}
				time.Sleep(1 * time.Second)
			}
		}

		req := clientset.CoreV1().RESTClient().Post().Resource("pods").Name(podName).Namespace(podNamespace).SubResource("exec")
		req.VersionedParams(&apiv1.PodExecOptions{
			TypeMeta:  metav1.TypeMeta{},
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
			Container: "sshbox",
			Command:   []string{"zsh"},
		}, scheme.ParameterCodec)

		exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
		if err != nil {
			io.WriteString(s, fmt.Sprint("Error during the command executer", err))
			return
		}

		err = exec.Stream(remotecommand.StreamOptions{
			Stdin:  s,
			Stdout: s,
			Stderr: s,
		})

		if err != nil {
			io.WriteString(s, err.Error()+"\n")
			return
		}

		io.WriteString(s, "Quitted!\n")

	})

	publicKeyOption := ssh.PublicKeyAuth(func(ctx ssh.Context, key ssh.PublicKey) bool {
		return true // allow all keys, or use ssh.KeysEqual() to compare against known keys
	})
	makeSSHKeyPair(".id_server.pub", ".id_server")
	log.Fatal(ssh.ListenAndServe(":22", nil, publicKeyOption, ssh.HostKeyFile(".id_server")))

}

//https://stackoverflow.com/a/34347463
func makeSSHKeyPair(pubKeyPath, privateKeyPath string) error {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return err
	}

	// generate and write private key as PEM
	privateKeyFile, err := os.Create(privateKeyPath)
	if err != nil {
		return err
	}
	defer privateKeyFile.Close()
	privateKeyPEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}
	if err := pem.Encode(privateKeyFile, privateKeyPEM); err != nil {
		return err
	}

	// generate and write public key
	pub, err := sshp.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(pubKeyPath, sshp.MarshalAuthorizedKey(pub), 0655)
}
