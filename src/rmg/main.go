package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	//"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"log"
	"os"
	"time"
)

func main() {
	environment := flag.String("env", "", "environment to perform the switch")
	path := flag.String("path", "", "project path")
	elb_type := flag.String("elb-type", "", "the elb type")
	flag.Parse()
	sess, err := GetAWSSession()
	if err != nil {
		fmt.Println("error getting session:", err)
		os.Exit(1)
	}
	elbService := elbv2.New(sess)
	fmt.Printf("Looking for project %s, in %s\r\n", *path, *environment)
	selectedSourceGroups, selectedTargetGroups, err := getSourceAndTargetGroups(*environment, *path, *elb_type, sess)
	if len(selectedSourceGroups) == 0 || len(selectedTargetGroups) == 0 {
		fmt.Println("target or source groups have no members, can't switch")
		os.Exit(1)
	}
	if err != nil {
		fmt.Println("can't get source/target groups", err)
		os.Exit(1)
	}
	if len(selectedSourceGroups) != 1 {
		fmt.Printf("There are %d source group. can only use one.\r\n", len(selectedSourceGroups))
		os.Exit(1)
	}
	sourceHealthCheck, err := elbService.DescribeTargetHealth(&elbv2.DescribeTargetHealthInput{
		TargetGroupArn: selectedSourceGroups[0].TargetGroupArn,
	})

	if nil != err {
		fmt.Println("error getting the source instances")
	}

	targetRegisterRequest := []*elbv2.TargetDescription{}
	fmt.Println("source groups:")
	for _, currentTargetGroup := range selectedSourceGroups {
		targetDescription, _ := elbService.DescribeTargetHealth(&elbv2.DescribeTargetHealthInput{
			TargetGroupArn: currentTargetGroup.TargetGroupArn,
		})
		for ord, desc := range targetDescription.TargetHealthDescriptions {
			targetRegisterRequest = append(targetRegisterRequest, desc.Target)
			fmt.Printf("%d. %s - %s\r\n", ord, *desc.Target.Id, *desc.TargetHealth.State)
		}
	}
	for _, currentTargetGroup := range selectedTargetGroups {
		elbService.RegisterTargets(&elbv2.RegisterTargetsInput{
			TargetGroupArn: currentTargetGroup.TargetGroupArn,
			Targets:        targetRegisterRequest,
		})
	}

	for _, currentTargetGroup := range selectedTargetGroups {
		approuvedTargetGroupsNumber := 0
		sleepers := 0
		targetDescription, _ := elbService.DescribeTargetHealth(&elbv2.DescribeTargetHealthInput{
			TargetGroupArn: currentTargetGroup.TargetGroupArn,
		})
		healthCount := 0
		for _, currentState := range targetDescription.TargetHealthDescriptions {
			if *currentState.TargetHealth.State == "healthy" {
				healthCount++
			}
		}
		if healthCount != len(targetDescription.TargetHealthDescriptions) && sleepers < 10 {
			log.Println("still waiting for the healthcheck")
			time.Sleep(time.Second * 5)
			sleepers++
		} else if sleepers >= 10 {
			log.Println("Waiting too long for healthcheck to be finished")
			os.Exit(1)
		} else if healthCount == len(targetDescription.TargetHealthDescriptions) {
			approuvedTargetGroupsNumber++
		}
		if approuvedTargetGroupsNumber == len(selectedSourceGroups) {
			log.Println("Done inserting.")
			break
		}
	}

	log.Println("Done")
}

// GetAWSSession -  creates an aws session
func GetAWSSession() (*session.Session, error) {
	sess, err := session.NewSession(&aws.Config{Region: aws.String("us-east-1")})
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return nil, errors.New("can't create aws session")
	}
	return sess, nil
}
