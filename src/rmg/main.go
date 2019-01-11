package main

import (
	"flag"
	"fmt"

	//"github.com/aws/aws-sdk-go/service/ec2"
	"log"
	"os"
)

func main() {
	environment := flag.String("env", "", "environment to perform the switch")
	path := flag.String("path", "", "project path")
	elbType := flag.String("elb-type", "", "the elb type")
	removeSource := flag.String("remove-source", "no", "remove the source instances from the source target group")
	flag.Parse()
	sess, err := GetAWSSession()
	if err != nil {
		fmt.Println("error getting session:", err)
		os.Exit(1)
	}
	log.Printf("Looking for project %s, in %s\r\n", *path, *environment)
	selector := &TargetGroupSelector{}
	if err = selector.init(sess, *environment, *path, *elbType); err != nil {
		log.Fatal(err)
	}
	log.Println("selector initialized, scanning target groups...")
	if err = selector.checkTargetGroupsForMatch(); err != nil {
		log.Fatal(err)
	}
	log.Printf("done scanning: \r\n%v\r\n", selector.GetTargetGroupsName())
	instancesSwitcher := InstancesSwitcher{}
	log.Println("switching")
	if err = instancesSwitcher.Init(sess, selector.SelectedSourceGroups, selector.SelectedTargetGroups); err != nil {
		log.Fatal(err)
	}
	if *removeSource == "yes" {
		err = instancesSwitcher.SwitchInstancesAndRemoveOldSources()
	} else {
		err = instancesSwitcher.SwitchInstances()
	}
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Done")
}
