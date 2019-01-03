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
	allTargetGroups, err := elbService.DescribeTargetGroups(&elbv2.DescribeTargetGroupsInput{})
	if err != nil {
		return nil, nil, err
	}
	for _, currentTargetGroupt := range allTargetGroups.TargetGroups {
		targetGroupTags, err := elbService.DescribeTags(&elbv2.DescribeTagsInput{
			ResourceArns: []*string{currentTargetGroupt.TargetGroupArn},
		})
		if err != nil {
			continue
		}

	}
	return selectedSourceGroups, selectedTargetGroups, nil
}

func checkTargetGroupTagsForMatch(targetGroupTags *elbv2.DescribeTagsOutput, environment, path, elbType string) {
	tagNameValues := []string{"Environment", "Elb-Type", "Path-Name", "Release"}
	tagValues := []string{environment, elbType, path, "yes"}
	numberOfMatchingTags := 0
	isAReleaseTargetGroup := false
	for _, tagDescription := range targetGroupTags.TagDescriptions {
		if numberOfMatchingTags == 4 {
			break
		}
		for _, tagMeta := range tagDescription.Tags {
			for i, tagName := range tagNameValues {
				if *tagMeta.Key == "Release" {
					numberOfMatchingTags++
					if *tagMeta.Value == "yes" {
						isAReleaseTargetGroup = true
					} else {
						isAReleaseTargetGroup = false
					}
				} else if *tagMeta.Key == tagName && *tagMeta.Value == tagValues[i] {
					numberOfMatchingTags++
				}
				if numberOfMatchingTags == len(tagNameValues) {
					if !isAReleaseTargetGroup {
						selectedSourceGroups = append(selectedSourceGroups, currentTargetGroupt)
					} else {
						selectedTargetGroups = append(selectedTargetGroups, currentTargetGroupt)
					}
				}
			}
		}
	}
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
