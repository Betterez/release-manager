package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/elbv2"
)

func createInstanceIDChecker(sourceTargetGroup *elbv2.TargetGroup) (func(string) bool, error) {
	sess, err := GetAWSSession()
	if nil != err {
		return nil, err
	}
	elbService := elbv2.New(sess)
	allTargetsIds := []string{}
	descriptions, err := elbService.DescribeTargetHealth(&elbv2.DescribeTargetHealthInput{
		TargetGroupArn: sourceTargetGroup.TargetGroupArn,
	})
	if nil != err {
		return nil, err
	}
	for _, description := range descriptions.TargetHealthDescriptions {
		allTargetsIds = append(allTargetsIds, *description.Target.Id)
	}

	return func(instanceId string) bool {
		for _, currentInstanceID := range allTargetsIds {
			if currentInstanceID == instanceId {
				return true
			}
		}
		return false
	}, nil
}

func removeOldInstancesFrom(sourceTargetGroup *elbv2.TargetGroup, targetTargetGroups []*elbv2.TargetGroup) error {
	sess, err := GetAWSSession()
	if nil != err {
		return err
	}
	elbService := elbv2.New(sess)
	idChecker, err := createInstanceIDChecker(sourceTargetGroup)
	if nil != err {
		return err
	}
	for _, currentTargetTargetGroup := range targetTargetGroups {
		targetsToRemove := []*elbv2.TargetDescription{}
		descriptions, err := elbService.DescribeTargetHealth(&elbv2.DescribeTargetHealthInput{
			TargetGroupArn: currentTargetTargetGroup.TargetGroupArn,
		})
		if nil != err {
			return err
		}
		for _, description := range descriptions.TargetHealthDescriptions {
			if !idChecker(*description.Target.Id) {
				targetsToRemove = append(targetsToRemove, description.Target)
			}
		}
		fmt.Printf("Removing %d instances from %s\r\n", len(targetsToRemove), *currentTargetTargetGroup.TargetGroupName)
		if len(targetsToRemove) > 0 {
			_, err = elbService.DeregisterTargets(&elbv2.DeregisterTargetsInput{
				TargetGroupArn: currentTargetTargetGroup.TargetGroupArn,
				Targets:        targetsToRemove,
			})
		}
		if nil != err {
			return err
		}
	}
	return nil
}
