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
	log.Println("done scanning")
	instancesSwitcher := InstancesSwitcher{}
	if err = instancesSwitcher.Init(sess, selector.SelectedSourceGroups, selector.SelectedTargetGroups); err != nil {
		// if err = instancesSwitcher.Init(sess, []*elbv2.TargetGroup{
		// 	{
		// 		TargetGroupArn: aws.String("arn:aws:elasticloadbalancing:us-east-1:109387325558:targetgroup/ngtg-stgng-notifications/bde5f8e46e4c88f4"),
		// 	}}, []*elbv2.TargetGroup{
		// 	{
		// 		TargetGroupArn: aws.String("arn:aws:elasticloadbalancing:us-east-1:109387325558:targetgroup/staging-notifications-in-tg/2982b148db63dd02"),
		// 	},
		// }); err != nil {
		log.Fatal(err)
	}
	if err = instancesSwitcher.getInstancesInGroups(); err != nil {
		log.Fatal(err)
	}
	fmt.Println(instancesSwitcher.sourceInstancesDescriptions)
	fmt.Println(instancesSwitcher.targetInstancesMapDescriptions)
	log.Println("Done")
}
