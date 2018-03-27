package main

import (
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
