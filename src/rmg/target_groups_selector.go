package main

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elbv2"
)

// TargetGroupSelector - filter target groups based on tags
type TargetGroupSelector struct {
	selectedSourceGroups []*elbv2.TargetGroup
	selectedTargetGroups []*elbv2.TargetGroup
	allTargetGroups      *elbv2.DescribeTargetGroupsOutput
	awsSession           *session.Session
}

func (tgs *TargetGroupSelector) init(sess *session.Session) {
	tgs.awsSession = sess
	tgs.selectedSourceGroups = make([]*elbv2.TargetGroup, 0)
	tgs.selectedTargetGroups = make([]*elbv2.TargetGroup, 0)
}

func (tgs *TargetGroupSelector) getAllTargetGroups() error {
	elbService := elbv2.New(tgs.awsSession)
	var err error
	tgs.allTargetGroups, err = elbService.DescribeTargetGroups(&elbv2.DescribeTargetGroupsInput{})
	if err != nil {
		return err
	}
	return nil
}

func (tgs *TargetGroupSelector) checkTargetGroupsForMatch() error {
	elbService := elbv2.New(tgs.awsSession)
	for _, currentTargetGroupt := range tgs.allTargetGroups.TargetGroups {
		targetGroupTags, err := elbService.DescribeTags(&elbv2.DescribeTagsInput{
			ResourceArns: []*string{currentTargetGroupt.TargetGroupArn},
		})
		if err != nil {
			return err
		}
		tgs.checkTargetGroupTagsForMatch(targetGroupTags)
	}
	return nil
}

func (tgs *TargetGroupSelector) checkTargetGroupTagsForMatch(tagsInfo *elbv2.DescribeTagsOutput) error {

	return nil
}
