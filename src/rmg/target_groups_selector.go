package main

import (
	"errors"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elbv2"
)

// TargetGroupSelector - filter target groups based on tags
type TargetGroupSelector struct {
	// SelectedSourceGroups - target groups that are release and will be used to get new instances
	SelectedSourceGroups []*elbv2.TargetGroup
	// SelectedTargetGroups - target groups that are not release, and will be used to put the new instances
	SelectedTargetGroups         []*elbv2.TargetGroup
	awsSession                   *session.Session
	allTargetGroupsForTheAccount *elbv2.DescribeTargetGroupsOutput
	environment                  string
	path                         string
	elbType                      string
}

// TargetGroupSearchResult - results from target group search
type TargetGroupSearchResult int

const (
	// TargetGroupNotFound - target group not found
	TargetGroupNotFound TargetGroupSearchResult = iota
	// TargetGroupFoundNonRelease -source (not release) tg
	TargetGroupFoundNonRelease
	// TargetGroupFoundRelease - release tg
	TargetGroupFoundRelease
)

func (tgs *TargetGroupSelector) init(sess *session.Session, environment, path, elbType string) error {
	if sess == nil {
		return errors.New("Bad session object")
	}
	tgs.awsSession = sess
	tgs.SelectedSourceGroups = make([]*elbv2.TargetGroup, 0)
	tgs.SelectedTargetGroups = make([]*elbv2.TargetGroup, 0)
	tgs.path = path
	tgs.environment = environment
	tgs.elbType = elbType
	return nil
}

func (tgs *TargetGroupSelector) getallTargetGroupsForTheAccount() error {
	elbService := elbv2.New(tgs.awsSession)
	var err error
	tgs.allTargetGroupsForTheAccount, err = elbService.DescribeTargetGroups(&elbv2.DescribeTargetGroupsInput{})
	if err != nil {
		return err
	}
	return nil
}

func (tgs *TargetGroupSelector) checkTargetGroupsForMatch() error {
	tgs.getallTargetGroupsForTheAccount()
	elbService := elbv2.New(tgs.awsSession)
	for _, currentTargetGroupt := range tgs.allTargetGroupsForTheAccount.TargetGroups {
		targetGroupTags, err := elbService.DescribeTags(&elbv2.DescribeTagsInput{
			ResourceArns: []*string{currentTargetGroupt.TargetGroupArn},
		})
		if err != nil {
			return err
		}
		switch tgs.checkTargetGroupTagsForMatch(targetGroupTags) {
		case TargetGroupFoundRelease:
			tgs.SelectedSourceGroups = append(tgs.SelectedTargetGroups, currentTargetGroupt)
			break
		case TargetGroupFoundNonRelease:
			tgs.SelectedTargetGroups = append(tgs.SelectedSourceGroups, currentTargetGroupt)
			break
		default:
			break
		}
	}
	return nil
}

func (tgs *TargetGroupSelector) checkTargetGroupTagsForMatch(targetGroupTags *elbv2.DescribeTagsOutput) TargetGroupSearchResult {
	numberOfMatchingTags := 0
	isAReleaseTargetGroup := false
	for _, tagDescription := range targetGroupTags.TagDescriptions {
		for _, currentTagData := range tagDescription.Tags {
			switch *currentTagData.Key {
			case "Release":
				if *currentTagData.Value == "yes" {
					isAReleaseTargetGroup = true
					numberOfMatchingTags++
				}
				if *currentTagData.Value == "no" {
					isAReleaseTargetGroup = false
					numberOfMatchingTags++
				}
				break
			case "Environment":
				if *currentTagData.Value == tgs.environment {
					numberOfMatchingTags++
				}
				break
			case "Elb-Type":
				if *currentTagData.Value == tgs.elbType {
					numberOfMatchingTags++
				}
				break
			case "Path-Name":
				if *currentTagData.Value == tgs.path {
					numberOfMatchingTags++
				}
				break
			}
		}
	}
	if numberOfMatchingTags == 4 {
		if isAReleaseTargetGroup {
			return TargetGroupFoundRelease
		}
		return TargetGroupFoundNonRelease
	}
	return TargetGroupNotFound
}
