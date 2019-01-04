package main

import (
	"errors"
	"fmt"

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

func (thisSelector *TargetGroupSelector) init(sess *session.Session, environment, path, elbType string) error {
	if sess == nil {
		return errors.New("Bad session object")
	}
	thisSelector.awsSession = sess
	thisSelector.SelectedSourceGroups = make([]*elbv2.TargetGroup, 0)
	thisSelector.SelectedTargetGroups = make([]*elbv2.TargetGroup, 0)
	thisSelector.path = path
	thisSelector.environment = environment
	thisSelector.elbType = elbType
	return nil
}

func (thisSelector *TargetGroupSelector) getallTargetGroupsForTheAccount() error {
	elbService := elbv2.New(thisSelector.awsSession)
	var err error
	thisSelector.allTargetGroupsForTheAccount, err = elbService.DescribeTargetGroups(&elbv2.DescribeTargetGroupsInput{})
	if err != nil {
		return err
	}
	return nil
}

func (thisSelector *TargetGroupSelector) checkTargetGroupsForMatch() error {
	thisSelector.getallTargetGroupsForTheAccount()
	elbService := elbv2.New(thisSelector.awsSession)
	for _, currentTargetGroupt := range thisSelector.allTargetGroupsForTheAccount.TargetGroups {
		targetGroupTags, err := elbService.DescribeTags(&elbv2.DescribeTagsInput{
			ResourceArns: []*string{currentTargetGroupt.TargetGroupArn},
		})
		if err != nil {
			return err
		}
		switch thisSelector.checkTargetGroupTagsForMatch(targetGroupTags) {
		case TargetGroupFoundRelease:
			fmt.Printf("adding %s to release(source)\r\n", *currentTargetGroupt.TargetGroupName)
			thisSelector.SelectedSourceGroups = append(thisSelector.SelectedTargetGroups, currentTargetGroupt)
			break
		case TargetGroupFoundNonRelease:
			fmt.Printf("adding %s to none release(target)\r\n", *currentTargetGroupt.TargetGroupName)
			thisSelector.SelectedTargetGroups = append(thisSelector.SelectedSourceGroups, currentTargetGroupt)
			break
		default:
			break
		}
	}
	fmt.Printf("done scanning: \r\n%v\r\n", thisSelector.GetTargetGroupsName())
	return nil
}

func (thisSelector *TargetGroupSelector) checkTargetGroupTagsForMatch(targetGroupTags *elbv2.DescribeTagsOutput) TargetGroupSearchResult {
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
				if *currentTagData.Value == thisSelector.environment {
					numberOfMatchingTags++
				}
				break
			case "Elb-Type":
				if *currentTagData.Value == thisSelector.elbType {
					numberOfMatchingTags++
				}
				break
			case "Path-Name":
				if *currentTagData.Value == thisSelector.path {
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

// GetTargetGroupsName - returns a map with the target groups names
func (thisSelector *TargetGroupSelector) GetTargetGroupsName() map[string][]string {
	result := make(map[string][]string, 0)
	for _, sourceGroup := range thisSelector.SelectedSourceGroups {
		result["source"] = append(result["source"], *sourceGroup.TargetGroupName)
	}
	for _, targetGroup := range thisSelector.SelectedTargetGroups {
		result["target"] = append(result["target"], *targetGroup.TargetGroupName)
	}
	return result
}
