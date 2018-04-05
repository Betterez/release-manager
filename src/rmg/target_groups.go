package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elbv2"
)

func getSourceAndTargetGroups(environment, path, elbType string, sess *session.Session) ([]*elbv2.TargetGroup, []*elbv2.TargetGroup, error) {
	selectedSourceGroups := []*elbv2.TargetGroup{}
	selectedTargetGroups := []*elbv2.TargetGroup{}
	elbService := elbv2.New(sess)
	targetGroups, err := elbService.DescribeTargetGroups(&elbv2.DescribeTargetGroupsInput{})
	if err != nil {
		return nil, nil, err
	}
	for _, currentTargetGroupt := range targetGroups.TargetGroups {
		numberOfPasses := 0
		releaseTg := false
		tags, err := elbService.DescribeTags(&elbv2.DescribeTagsInput{
			ResourceArns: []*string{currentTargetGroupt.TargetGroupArn},
		})
		if err != nil {
			continue
		}
		for _, tagDescription := range tags.TagDescriptions {
			if numberOfPasses == 4 {
				break
			}
			for _, tagMeta := range tagDescription.Tags {
				if *tagMeta.Key == "Environment" && *tagMeta.Value == environment {
					numberOfPasses++
				}
				if *tagMeta.Key == "Elb-Type" && *tagMeta.Value == elbType {
					numberOfPasses++
				}
				if *tagMeta.Key == "Path-Name" && *tagMeta.Value == path {
					numberOfPasses++
				}
				if *tagMeta.Key == "Release" {
					numberOfPasses++
					if *tagMeta.Value == "yes" {
						releaseTg = true
					} else {
						releaseTg = false
					}
				}
				if numberOfPasses == 4 {
					if releaseTg {
						selectedSourceGroups = append(selectedSourceGroups, currentTargetGroupt)
					} else {
						selectedTargetGroups = append(selectedTargetGroups, currentTargetGroupt)
					}
				}
			}
		}
	}
	return selectedSourceGroups, selectedTargetGroups, nil
}

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
