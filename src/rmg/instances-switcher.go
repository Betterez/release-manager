package main

import (
	"errors"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elbv2"
)

// InstancesSwitcher - deploy instances from the source group to the target one
type InstancesSwitcher struct {
	awsSession                  *session.Session
	SelectedSourceGroups        []*elbv2.TargetGroup
	SelectedTargetGroups        []*elbv2.TargetGroup
	sourceInstancesDescriptions []*elbv2.TargetDescription
	// TODO: do we need this? this might be removed
	targetInstancesMapDescriptions map[string][]*elbv2.TargetDescription
}

// Init - initialize
func (is *InstancesSwitcher) Init(sess *session.Session, sourceTargetGroups, targetTargetGroups []*elbv2.TargetGroup) error {
	if sess == nil || sourceTargetGroups == nil || targetTargetGroups == nil {
		return errors.New("parameters nil error")
	}
	if len(sourceTargetGroups) == 0 || len(targetTargetGroups) == 0 {
		return errors.New("empty source or target group")
	}
	if len(sourceTargetGroups) > 1 {
		return errors.New("source groups cna only contain 1 group")
	}
	is.awsSession = sess
	is.SelectedSourceGroups = sourceTargetGroups
	is.SelectedTargetGroups = targetTargetGroups
	is.targetInstancesMapDescriptions = make(map[string][]*elbv2.TargetDescription, 0)
	return nil
}

// SwitchInstances - switch instances from the source group to the rtarget groups
func (is *InstancesSwitcher) SwitchInstances() error {
	if len(is.sourceInstancesDescriptions) == 0 {
		return errors.New("no source instances")
	}
	for _, targetGroup := range is.SelectedTargetGroups {
		// TODO: the source parameter can be removed
		is.registerInstancesWithTargetGroup(targetGroup, is.sourceInstancesDescriptions)
		is.removeOldInstancesFromTargetGroup(targetGroup, is.sourceInstancesDescriptions)
	}
	return nil
}

func (is *InstancesSwitcher) removeOldInstancesFromTargetGroup(targetTargetGroup *elbv2.TargetGroup, sourceInstances []*elbv2.TargetDescription) error {
	elbService := elbv2.New(is.awsSession)
	var targetInstancesDescription, instancesToBeRemoved []*elbv2.TargetDescription
	getInstancesDescriptionForGroups([]*elbv2.TargetGroup{targetTargetGroup}, &targetInstancesDescription, is.awsSession)
	sourceInstancesMap := mapInstancesByID(sourceInstances)
	for _, instanceDescription := range targetInstancesDescription {
		if sourceInstancesMap[*instanceDescription.Id] > 0 {
			continue
		}
		instancesToBeRemoved = append(instancesToBeRemoved, instanceDescription)
	}
	if len(instancesToBeRemoved) > 0 {
		elbService.DeregisterTargets(&elbv2.DeregisterTargetsInput{
			TargetGroupArn: targetTargetGroup.TargetGroupArn,
			Targets:        instancesToBeRemoved,
		})
	}
	return nil
}

func mapInstancesByID(instances []*elbv2.TargetDescription) map[string]int {
	result := make(map[string]int, 0)
	for _, instanceData := range instances {
		result[*instanceData.Id]++
	}
	return result
}

func (is *InstancesSwitcher) registerInstancesWithTargetGroup(targetTargetGroup *elbv2.TargetGroup, instances []*elbv2.TargetDescription) error {
	elbService := elbv2.New(is.awsSession)
	elbService.RegisterTargets(&elbv2.RegisterTargetsInput{
		TargetGroupArn: targetTargetGroup.TargetGroupArn,
		Targets:        instances,
	})
	healthyInstances := 0
	for {
		time.Sleep(5 * time.Second)
		instancesHealth, err := elbService.DescribeTargetHealth(&elbv2.DescribeTargetHealthInput{
			TargetGroupArn: targetTargetGroup.TargetGroupArn,
			Targets:        instances,
		})
		if err != nil {
			return err
		}
		for _, instanceHealthDescription := range instancesHealth.TargetHealthDescriptions {
			if *instanceHealthDescription.TargetHealth.State == "healthy" {
				healthyInstances++
			}
		}
		if healthyInstances == len(instances) {
			break
		}
		healthyInstances = 0
	}
	return nil
}

func (is *InstancesSwitcher) getInstancesInGroups() error {
	is.sourceInstancesDescriptions = make([]*elbv2.TargetDescription, 0)
	if err := getInstancesDescriptionForGroups(is.SelectedSourceGroups, &is.sourceInstancesDescriptions, is.awsSession); err != nil {
		return err
	}
	var targetGroupDescription []*elbv2.TargetDescription
	for _, targetGroup := range is.SelectedTargetGroups {
		if err := getInstancesDescriptionForGroups([]*elbv2.TargetGroup{targetGroup},
			&targetGroupDescription,
			is.awsSession); err != nil {
			return err
		}
		is.targetInstancesMapDescriptions[*targetGroup.TargetGroupName] = targetGroupDescription
	}

	return nil
}

func getInstancesDescriptionForGroups(targetGroups []*elbv2.TargetGroup,
	instancesDescriptions *[]*elbv2.TargetDescription,
	awsSession *session.Session) error {
	elbService := elbv2.New(awsSession)

	for _, currentTargetGroup := range targetGroups {
		targetDescription, err := elbService.DescribeTargetHealth(&elbv2.DescribeTargetHealthInput{
			TargetGroupArn: currentTargetGroup.TargetGroupArn,
		})
		if err != nil {
			return err
		}
		for _, desc := range targetDescription.TargetHealthDescriptions {
			if *desc.TargetHealth.State == "healthy" {
				*instancesDescriptions = append(*instancesDescriptions, desc.Target)
			}
		}
	}
	return nil
}
