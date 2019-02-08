package main

import (
	"errors"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elbv2"
)

const (
	MaximumWaitingLoops = 15
)

// InstancesSwitcher - deploy instances from the source group to the target one
type InstancesSwitcher struct {
	awsSession                  *session.Session
	SelectedSourceTargetGroups  []*elbv2.TargetGroup
	SelectedTargetTargetGroups  []*elbv2.TargetGroup
	sourceInstancesDescriptions []*elbv2.TargetDescription
	// TODO: do we need this? ththisSwitcher might be removed
	targetInstancesMapDescriptions map[string][]*elbv2.TargetDescription
}

// Init - initialize
func (thisSwitcher *InstancesSwitcher) Init(sess *session.Session, sourceTargetGroups, targetTargetGroups []*elbv2.TargetGroup) error {
	if sess == nil || sourceTargetGroups == nil || targetTargetGroups == nil {
		return errors.New("parameters nil error")
	}
	if len(sourceTargetGroups) == 0 || len(targetTargetGroups) == 0 {
		return errors.New("empty source or target group")
	}
	if len(sourceTargetGroups) > 1 {
		return errors.New("source groups cna only contain 1 group")
	}
	thisSwitcher.awsSession = sess
	thisSwitcher.SelectedSourceTargetGroups = sourceTargetGroups
	thisSwitcher.SelectedTargetTargetGroups = targetTargetGroups
	thisSwitcher.targetInstancesMapDescriptions = make(map[string][]*elbv2.TargetDescription, 0)
	thisSwitcher.sourceInstancesDescriptions = make([]*elbv2.TargetDescription, 0)
	return nil
}

// SwitchInstances - switch instances from the source group to the rtarget groups
func (thisSwitcher *InstancesSwitcher) SwitchInstances() error {
	if err := thisSwitcher.getInstancesInGroupsIfNeeded(); err != nil {
		return err
	}
	if len(thisSwitcher.sourceInstancesDescriptions) == 0 {
		return errors.New("no source instances")
	}
	for _, targetGroup := range thisSwitcher.SelectedTargetTargetGroups {
		thisSwitcher.registerInstancesWithTargetGroup(targetGroup)
		thisSwitcher.removeOldInstancesFromTargetGroup(targetGroup)
	}
	return nil
}

// SwitchInstancesAndRemoveOldSources - switching instances and removing the old ones
func (thisSwitcher *InstancesSwitcher) SwitchInstancesAndRemoveOldSources() error {
	err := thisSwitcher.SwitchInstances()
	if err != nil {
		return err
	}
	thisSwitcher.cleanSourceTargetGroupFromOldInstances()
	return nil
}

func (thisSwitcher *InstancesSwitcher) registerInstancesWithTargetGroup(targetTargetGroup *elbv2.TargetGroup) error {
	elbService := elbv2.New(thisSwitcher.awsSession)
	elbService.RegisterTargets(&elbv2.RegisterTargetsInput{
		TargetGroupArn: targetTargetGroup.TargetGroupArn,
		Targets:        thisSwitcher.sourceInstancesDescriptions,
	})
	healthyInstances := 0
	loopCounter := 0
	for {
		time.Sleep(5 * time.Second)
		instancesHealth, err := elbService.DescribeTargetHealth(&elbv2.DescribeTargetHealthInput{
			TargetGroupArn: targetTargetGroup.TargetGroupArn,
			Targets:        thisSwitcher.sourceInstancesDescriptions,
		})
		if err != nil {
			return err
		}
		for _, instanceHealthDescription := range instancesHealth.TargetHealthDescriptions {
			if *instanceHealthDescription.TargetHealth.State == "healthy" || *instanceHealthDescription.TargetHealth.State == "unused" {
				healthyInstances++
			}
		}
		if healthyInstances == len(thisSwitcher.sourceInstancesDescriptions) {
			break
		}
		healthyInstances = 0
		loopCounter++
		if loopCounter >= MaximumWaitingLoops {
			return errors.New("time out waiting for instance")
		}
	}
	return nil
}

func (thisSwitcher *InstancesSwitcher) removeOldInstancesFromTargetGroup(targetTargetGroup *elbv2.TargetGroup) error {
	elbService := elbv2.New(thisSwitcher.awsSession)
	var err error
	var targetInstancesDescription, instancesToBeRemoved []*elbv2.TargetDescription
	if targetInstancesDescription, err = getInstancesDescriptionForGroups([]*elbv2.TargetGroup{targetTargetGroup}, thisSwitcher.awsSession); err != nil {
		return err
	}
	sourceInstancesMap := mapInstancesByID(thisSwitcher.sourceInstancesDescriptions)
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

func (thisSwitcher *InstancesSwitcher) cleanSourceTargetGroupFromOldInstances() {
	elbService := elbv2.New(thisSwitcher.awsSession)
	elbService.DeregisterTargets(&elbv2.DeregisterTargetsInput{
		TargetGroupArn: thisSwitcher.SelectedSourceTargetGroups[0].TargetGroupArn,
		Targets:        thisSwitcher.sourceInstancesDescriptions,
	})
}

func mapInstancesByID(instances []*elbv2.TargetDescription) map[string]int {
	result := make(map[string]int, 0)
	for _, instanceData := range instances {
		result[*instanceData.Id]++
	}
	return result
}

func (thisSwitcher *InstancesSwitcher) getInstancesInGroupsIfNeeded() error {
	if len(thisSwitcher.sourceInstancesDescriptions) == 0 {
		return thisSwitcher.getInstancesInGroups()
	}
	return nil
}

func (thisSwitcher *InstancesSwitcher) getInstancesInGroups() error {
	var err error

	if thisSwitcher.sourceInstancesDescriptions, err = getInstancesDescriptionForGroups(thisSwitcher.SelectedSourceTargetGroups, thisSwitcher.awsSession); err != nil {
		return err
	}
	var targetGroupDescription []*elbv2.TargetDescription
	for _, targetGroup := range thisSwitcher.SelectedTargetTargetGroups {
		if targetGroupDescription, err = getInstancesDescriptionForGroups([]*elbv2.TargetGroup{targetGroup}, thisSwitcher.awsSession); err != nil {
			return err
		}
		thisSwitcher.targetInstancesMapDescriptions[*targetGroup.TargetGroupName] = targetGroupDescription
	}
	return nil
}

func getInstancesDescriptionForGroups(targetGroups []*elbv2.TargetGroup,
	awsSession *session.Session) ([]*elbv2.TargetDescription, error) {
	elbService := elbv2.New(awsSession)
	var result []*elbv2.TargetDescription

	for _, currentTargetGroup := range targetGroups {
		targetDescription, err := elbService.DescribeTargetHealth(&elbv2.DescribeTargetHealthInput{
			TargetGroupArn: currentTargetGroup.TargetGroupArn,
		})
		if err != nil {
			return nil, err
		}
		for _, desc := range targetDescription.TargetHealthDescriptions {
			if *desc.TargetHealth.State == "healthy" || *desc.TargetHealth.State == "unused" {
				result = append(result, desc.Target)
			}
		}
	}
	return result, nil
}
