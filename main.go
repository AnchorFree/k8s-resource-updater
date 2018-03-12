// +build example

package main

import (
        "fmt"
        "os"
        "time"
	"net"
	"bufio"
	"os/signal"
	"syscall"

	"github.com/urfave/cli"

        "k8s.io/client-go/kubernetes"
        metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
        rest "k8s.io/client-go/rest"
        discovery "k8s.io/apimachinery/pkg/version"
        wait "k8s.io/apimachinery/pkg/util/wait"
	networkingv1 "k8s.io/api/networking/v1"
)

const (
        // High enough QPS to fit all expected use cases. QPS=0 is not set here, because
        // client code is overriding it.
        defaultQPS = 1e6
        // High enough Burst to fit all expected use cases. Burst=0 is not set here, because
        // client code is overriding it.
        defaultBurst = 1e6
)

func main() {
        app := cli.NewApp()
        app.Name = "k8s-resource-updater"
        app.Version = "0.0.1"

        cli.VersionFlag = cli.BoolFlag{
                Name:  "version, V",
                Usage: "print version number",
        }

        app.Flags = []cli.Flag{
                cli.StringFlag{
                        Name:   "k8s-resource-namespace",
                        Usage:  "Kubernetes cluster Namesace for updated resource",
                        EnvVar: "K8S_RESOURCE_NAMESPACE",
                },
                cli.StringFlag{
                        Name:   "k8s-resource-name",
                        Usage:  "Kubernetes cluster Name for updated resource",
                        EnvVar: "K8S_RESOURCE_NAME",
                },
                cli.StringFlag{
                        Name:   "k8s-resource-label",
                        Usage:  "Kubernetes cluster Label to filter updated resource",
                        EnvVar: "K8S_CUSTOM_LABEL_VALUE",
                },
                cli.StringFlag{
                        Name:   "k8s-file-to-read",
                        Usage:  "Consul templater filepath to read configuration to update kubernetes resource",
                        EnvVar: "K8S_FILE_TO_READ",
                },
		cli.StringFlag{
                        Name:   "verbose",
                        Usage:  "Verbose flag",
                        EnvVar: "VERBOSE",
                },
        }

        app.Commands = []cli.Command{
                {
                        Name:   "networkpolicy",
                        Usage:  "Update Kubernetes network policy with provided ips from file: Usage k8s-resource-updater networkpolicy",
                        Action: CmdRunNetworkPolicyUpdate,
                        Flags:  app.Flags,
                },
        }


	signal_chan := make(chan os.Signal, 1)
	signal.Notify(signal_chan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	exit_chan := make(chan int)
	
	go func() {
		for {
			s := <-signal_chan
			switch s {
			// kill -SIGHUP XXXX
			case syscall.SIGHUP:
				app.Run(os.Args)
			// kill -SIGINT XXXX or Ctrl+c
			case syscall.SIGINT:
				fmt.Println("Interrupt")
				exit_chan <- 0
			// kill -SIGTERM XXXX
			case syscall.SIGTERM:
				fmt.Println("force stop")
				exit_chan <- 0
			// kill -SIGQUIT XXXX
			case syscall.SIGQUIT:
				fmt.Println("stop and core dump")
				exit_chan <- 0
			default:
				fmt.Println("Unknown signal.")
				exit_chan <- 1
			}
		}
	}()

	app.Run(os.Args)

	code := <-exit_chan
	fmt.Println("Get exit code", code)
	os.Exit(code)
}

func readFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
  	defer file.Close()

  	var lines []string
	scanner := bufio.NewScanner(file)
  	for scanner.Scan() {
    		lines = append(lines, scanner.Text())
  	}

	return lines, scanner.Err()
}

func varifyCIDRList(cidr_list []string, verbose string) ([]string) {
        var result_cidr_list []string

        for i := range cidr_list {
                _, _, err := net.ParseCIDR(cidr_list[i])
                if err != nil {
                        if net.ParseIP(cidr_list[i]) == nil {
                                if verbose == "true" {
                                        fmt.Println("ERROR: IP '", cidr_list[i],"' is wrong, skipping it")
                                }
                        } else {
                                if verbose == "true" {
                                        fmt.Println("Not valid cidr ", cidr_list[i], " converting to single ip CIDR")
                                }
				result_cidr_list = append(result_cidr_list, cidr_list[i] + "/32")
                        }
                } else {
			result_cidr_list = append(result_cidr_list, cidr_list[i])
		}
        }

	return result_cidr_list
}

func CmdRunNetworkPolicyUpdate(c *cli.Context) error {
        verbose := c.GlobalString("verbose")
        k8s_namespace := "default"
        var k8s_resource_name string
        
	restircted_ips, err := readFile(c.GlobalString("k8s-file-to-read"))
	if err != nil {
        	fmt.Println("ERROR: ", err)
        	os.Exit(1)
    	}

	cidr_list := varifyCIDRList(restircted_ips,verbose)


	np_generated_from_ipblock := make([]networkingv1.NetworkPolicyPeer, len(cidr_list))

        for i := range cidr_list {
		np_generated_from_ipblock[i].IPBlock = &networkingv1.IPBlock{CIDR: cidr_list[i],}
        }

        if c.GlobalString("k8s-resource-namespace") != "" {
                k8s_namespace = c.GlobalString("k8s-resource-namespace")
        }
        if c.GlobalString("k8s-resource-name") != "" {
                k8s_resource_name = c.GlobalString("k8s-resource-name")
        } else {
                fmt.Println("ERROR: k8s-resource-name cannot be empty")
                os.Exit(1)
        }

        if verbose == "true" {
                fmt.Println("Updating NetworkPolicy ",k8s_resource_name," in Kubernetes on namespace ",k8s_namespace)
        }

        kubeClient, err := createApiserverClient()
        np, err := kubeClient.NetworkingV1().NetworkPolicies(k8s_namespace).Get(k8s_resource_name,metav1.GetOptions{})
        if err != nil {
		fmt.Println("ERROR: Cannot get networkpolicy to update")
                fmt.Println(err.Error())
        } else {
                np.Spec.Ingress = []networkingv1.NetworkPolicyIngressRule{
	                {From: np_generated_from_ipblock},
                }

                np, err = kubeClient.NetworkingV1().NetworkPolicies("default").Update(np)

                if err != nil {
                        fmt.Println(err.Error())
                }
        }

        return nil
}

func createApiserverClient() (*kubernetes.Clientset, error) {
        cfg, err := rest.InClusterConfig()
        if err != nil {
                fmt.Println("could not execute due to error:", err)
                return nil, err
        }

        cfg.QPS = defaultQPS
        cfg.Burst = defaultBurst
        cfg.ContentType = "application/vnd.kubernetes.protobuf"


        client, err := kubernetes.NewForConfig(cfg)
        if err != nil {
                fmt.Println("could not execute due to error:", err)
                return nil, err
        }

        var v *discovery.Info

        // In some environments is possible the client cannot connect the API server in the first request
        // https://github.com/kubernetes/ingress-nginx/issues/1968
        defaultRetry := wait.Backoff{
                Steps:    10,
                Duration: 1 * time.Second,
                Factor:   1.5,
                Jitter:   0.1,
        }

        var lastErr error
        retries := 0
        err = wait.ExponentialBackoff(defaultRetry, func() (bool, error) {
                v, err = client.Discovery().ServerVersion()
                if err == nil {
                        return true, nil
                }

                lastErr = err
                retries++
                return false, nil
        })

        // err is not null only if there was a timeout in the exponential backoff (ErrWaitTimeout)
        if err != nil {
                return nil, lastErr
        }



        return client, nil
}

