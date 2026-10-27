package label_service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	instance_model "github.com/EvolutionAPI/evolution-go/pkg/instance/model"
	label_model "github.com/EvolutionAPI/evolution-go/pkg/label/model"
	label_repository "github.com/EvolutionAPI/evolution-go/pkg/label/repository"
	logger_wrapper "github.com/EvolutionAPI/evolution-go/pkg/logger"
	"github.com/EvolutionAPI/evolution-go/pkg/utils"
	whatsmeow_service "github.com/EvolutionAPI/evolution-go/pkg/whatsmeow/service"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/appstate"
	"go.mau.fi/whatsmeow/types"
)

type LabelService interface {
	ChatLabel(data *ChatLabelStruct, instance *instance_model.Instance) error
	MessageLabel(data *MessageLabelStruct, instance *instance_model.Instance) error
	EditLabel(data *EditLabelStruct, instance *instance_model.Instance) error
	ChatUnlabel(data *ChatLabelStruct, instance *instance_model.Instance) error
	MessageUnlabel(data *MessageLabelStruct, instance *instance_model.Instance) error
	GetLabels(instance *instance_model.Instance) ([]label_model.Label, error)
	GetContactsByLabel(data *GetContactsByLabelStruct, instance *instance_model.Instance) (*LabelContactsResult, error)
	GetAllContactsByLabel(instance *instance_model.Instance) (*AllLabelContactsResult, error)
}

type labelService struct {
	clientPointer    map[string]*whatsmeow.Client
	whatsmeowService whatsmeow_service.WhatsmeowService
	labelRepository  label_repository.LabelRepository
	loggerWrapper    *logger_wrapper.LoggerManager
}

type ChatLabelStruct struct {
	JID     string `json:"jid"`
	LabelID string `json:"labelId"`
}

type MessageLabelStruct struct {
	JID       string `json:"jid"`
	LabelID   string `json:"labelId"`
	MessageID string `json:"messageId"`
}

type EditLabelStruct struct {
	LabelID string `json:"labelId"`
	Name    string `json:"name"`
	Color   int    `json:"color"`
	Deleted bool   `json:"deleted"`
}

func (l *labelService) ensureClientConnected(instanceId string) (*whatsmeow.Client, error) {
	client := l.clientPointer[instanceId]
	l.loggerWrapper.GetLogger(instanceId).LogInfo("[%s] Checking client connection status - Client exists: %v", instanceId, client != nil)

	if client == nil {
		l.loggerWrapper.GetLogger(instanceId).LogInfo("[%s] No client found, attempting to start new instance", instanceId)
		err := l.whatsmeowService.StartInstance(instanceId)
		if err != nil {
			l.loggerWrapper.GetLogger(instanceId).LogError("[%s] Failed to start instance: %v", instanceId, err)
			return nil, errors.New("no active session found")
		}

		l.loggerWrapper.GetLogger(instanceId).LogInfo("[%s] Instance started, waiting 2 seconds...", instanceId)
		time.Sleep(2 * time.Second)

		client = l.clientPointer[instanceId]
		l.loggerWrapper.GetLogger(instanceId).LogInfo("[%s] Checking new client - Exists: %v, Connected: %v",
			instanceId,
			client != nil,
			client != nil && client.IsConnected())

		if client == nil || !client.IsConnected() {
			l.loggerWrapper.GetLogger(instanceId).LogError("[%s] New client validation failed - Exists: %v, Connected: %v",
				instanceId,
				client != nil,
				client != nil && client.IsConnected())
			return nil, errors.New("no active session found")
		}
	} else if !client.IsConnected() {
		l.loggerWrapper.GetLogger(instanceId).LogError("[%s] Existing client is disconnected - Connected status: %v",
			instanceId,
			client.IsConnected())
		return nil, errors.New("client disconnected")
	}

	l.loggerWrapper.GetLogger(instanceId).LogInfo("[%s] Client successfully validated - Connected: %v", instanceId, client.IsConnected())
	return client, nil
}

func (l *labelService) ChatLabel(data *ChatLabelStruct, instance *instance_model.Instance) error {
	client, err := l.ensureClientConnected(instance.Id)
	if err != nil {
		return err
	}

	jid, ok := utils.ParseJID(data.JID)
	if !ok {
		l.loggerWrapper.GetLogger(instance.Id).LogError("[%s] error parse community jid", instance.Id)
		return errors.New("error parse community jid")
	}

	err = client.SendAppState(context.Background(), appstate.BuildLabelChat(
		jid,
		data.LabelID,
		true,
	))
	if err != nil {
		l.loggerWrapper.GetLogger(instance.Id).LogError("[%s] error label chat: %v", instance.Id, err)
		return err
	}

	return nil
}

func (l *labelService) MessageLabel(data *MessageLabelStruct, instance *instance_model.Instance) error {
	client, err := l.ensureClientConnected(instance.Id)
	if err != nil {
		return err
	}

	jid, ok := utils.ParseJID(data.JID)
	if !ok {
		l.loggerWrapper.GetLogger(instance.Id).LogError("[%s] error parse community jid", instance.Id)
		return errors.New("error parse community jid")
	}

	err = client.SendAppState(context.Background(), appstate.BuildLabelMessage(
		jid,
		data.LabelID,
		data.MessageID,
		true,
	))
	if err != nil {
		l.loggerWrapper.GetLogger(instance.Id).LogError("[%s] error label message: %v", instance.Id, err)
		return err
	}

	return nil
}

func (l *labelService) EditLabel(data *EditLabelStruct, instance *instance_model.Instance) error {
	client, err := l.ensureClientConnected(instance.Id)
	if err != nil {
		return err
	}

	err = client.SendAppState(context.Background(), appstate.BuildLabelEdit(
		data.LabelID,
		data.Name,
		int32(data.Color),
		data.Deleted,
	))
	if err != nil {
		l.loggerWrapper.GetLogger(instance.Id).LogError("[%s] error label message: %v", instance.Id, err)
		return err
	}

	return nil
}

func (l *labelService) ChatUnlabel(data *ChatLabelStruct, instance *instance_model.Instance) error {
	client, err := l.ensureClientConnected(instance.Id)
	if err != nil {
		return err
	}

	jid, ok := utils.ParseJID(data.JID)
	if !ok {
		l.loggerWrapper.GetLogger(instance.Id).LogError("[%s] error parse community jid", instance.Id)
		return errors.New("error parse community jid")
	}

	err = client.SendAppState(context.Background(), appstate.BuildLabelChat(
		jid,
		data.LabelID,
		false,
	))
	if err != nil {
		l.loggerWrapper.GetLogger(instance.Id).LogError("[%s] error label chat: %v", instance.Id, err)
		return err
	}

	return nil
}

func (l *labelService) MessageUnlabel(data *MessageLabelStruct, instance *instance_model.Instance) error {
	client, err := l.ensureClientConnected(instance.Id)
	if err != nil {
		return err
	}

	jid, ok := utils.ParseJID(data.JID)
	if !ok {
		l.loggerWrapper.GetLogger(instance.Id).LogError("[%s] error parse community jid", instance.Id)
		return errors.New("error parse community jid")
	}

	err = client.SendAppState(context.Background(), appstate.BuildLabelMessage(
		jid,
		data.LabelID,
		data.MessageID,
		false,
	))
	if err != nil {
		l.loggerWrapper.GetLogger(instance.Id).LogError("[%s] error label message: %v", instance.Id, err)
		return err
	}

	return nil
}

func (l *labelService) GetLabels(instance *instance_model.Instance) ([]label_model.Label, error) {
	_, err := l.ensureClientConnected(instance.Id)
	if err != nil {
		return nil, err
	}

	labels, err := l.labelRepository.GetAllLabelsByInstanceID(instance.Id)
	if err != nil {
		l.loggerWrapper.GetLogger(instance.Id).LogError("[%s] error fetching labels from database: %v", instance.Id, err)
		return nil, err
	}

	return labels, nil
}

type GetContactsByLabelStruct struct {
	LabelID       string `json:"labelId"`
	LabelName     string `json:"labelName"`
	Sync          *bool  `json:"sync,omitempty"`
	IncludeGroups bool   `json:"includeGroups,omitempty"`
}

type LabelContactInfo struct {
	JID          string `json:"jid"`
	PhoneNumber  string `json:"phoneNumber"`
	FirstName    string `json:"firstName"`
	FullName     string `json:"fullName"`
	PushName     string `json:"pushName"`
	BusinessName string `json:"businessName"`
	LID          string `json:"lid,omitempty"`
	IsGroup      bool   `json:"isGroup"`
}

type LabelContactsResult struct {
	Label    label_model.Label  `json:"label"`
	Contacts []LabelContactInfo `json:"contacts"`
	Total    int                `json:"total"`
	Synced   bool               `json:"synced"`
}

type AllLabelContactInfo struct {
	JID         string   `json:"jid"`
	PhoneNumber string   `json:"phoneNumber"`
	LID         string   `json:"lid,omitempty"`
	IsGroup     bool     `json:"isGroup"`
	Labels      []string `json:"labels"`
}

type AllLabelContactsResult struct {
	Contacts []AllLabelContactInfo `json:"contacts"`
	Total    int                   `json:"total"`
	Synced   bool                  `json:"synced"`
}

// labelSyncMu protects the temporary EmitAppStateEventsOnFullSync flag used
// while backfilling historical label associations from a full app state sync.
// SIMPLIFICATION: one global lock; per-instance locks if parallel syncs matter.
var labelSyncMu sync.Mutex

func cleanLabelName(name string) string {
	// WhatsApp prefixes some predefined labels with invisible direction marks.
	cleaned := strings.Map(func(r rune) rune {
		if r == '\u200e' || r == '\u200f' || r == '\u202a' || r == '\u202c' {
			return -1
		}
		return r
	}, name)
	return strings.TrimSpace(cleaned)
}

func normalizeLabelName(name string) string {
	return strings.ToLower(cleanLabelName(name))
}

func (l *labelService) resolveLabel(instanceID string, data *GetContactsByLabelStruct) (*label_model.Label, error) {
	if data.LabelID != "" {
		label, err := l.labelRepository.GetLabelByInstanceAndLabelID(instanceID, data.LabelID)
		if err != nil {
			return nil, err
		}
		return label, nil
	}

	if data.LabelName == "" {
		return nil, errors.New("labelId or labelName is required")
	}

	labels, err := l.labelRepository.GetAllLabelsByInstanceID(instanceID)
	if err != nil {
		return nil, err
	}

	target := normalizeLabelName(data.LabelName)
	for i := range labels {
		if normalizeLabelName(labels[i].LabelName) == target {
			return &labels[i], nil
		}
	}
	for i := range labels {
		if strings.Contains(normalizeLabelName(labels[i].LabelName), target) {
			return &labels[i], nil
		}
	}

	return nil, fmt.Errorf("label not found: %s", data.LabelName)
}

func (l *labelService) syncLabelAppState(client *whatsmeow.Client, instanceID string) error {
	labelSyncMu.Lock()
	defer labelSyncMu.Unlock()

	previous := client.EmitAppStateEventsOnFullSync
	client.EmitAppStateEventsOnFullSync = true
	defer func() {
		client.EmitAppStateEventsOnFullSync = previous
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	l.loggerWrapper.GetLogger(instanceID).LogInfo("[%s] Backfilling WhatsApp labels from regular app state", instanceID)
	if err := client.FetchAppState(ctx, appstate.WAPatchRegular, true, false); err != nil {
		l.loggerWrapper.GetLogger(instanceID).LogError("[%s] Failed to sync labels: %v", instanceID, err)
		return err
	}
	return nil
}

func (l *labelService) GetContactsByLabel(data *GetContactsByLabelStruct, instance *instance_model.Instance) (*LabelContactsResult, error) {
	client, err := l.ensureClientConnected(instance.Id)
	if err != nil {
		return nil, err
	}

	forceSync := data.Sync != nil && *data.Sync
	synced := false

	if forceSync {
		if err := l.syncLabelAppState(client, instance.Id); err != nil {
			return nil, err
		}
		synced = true
	}

	label, resolveErr := l.resolveLabel(instance.Id, data)
	if resolveErr != nil && !synced {
		// Labels may not have been synced to our database yet (e.g. first use
		// after deploying this feature). Pull them once before giving up.
		if syncErr := l.syncLabelAppState(client, instance.Id); syncErr == nil {
			synced = true
			label, resolveErr = l.resolveLabel(instance.Id, data)
		}
	}
	if resolveErr != nil {
		return nil, resolveErr
	}
	if label == nil {
		ref := data.LabelID
		if ref == "" {
			ref = data.LabelName
		}
		return nil, fmt.Errorf("label not found: %s", ref)
	}

	jids, err := l.labelRepository.GetChatJIDsByLabel(instance.Id, label.LabelID)
	if err != nil {
		return nil, err
	}

	if len(jids) == 0 && !synced {
		// Only backfill once per instance: if any label association is already
		// stored, this label is simply empty and another full sync is waste.
		existing, countErr := l.labelRepository.CountLabelChatsByInstanceID(instance.Id)
		if countErr != nil {
			return nil, countErr
		}
		if existing == 0 {
			if err := l.syncLabelAppState(client, instance.Id); err != nil {
				return nil, err
			}
			synced = true
			jids, err = l.labelRepository.GetChatJIDsByLabel(instance.Id, label.LabelID)
			if err != nil {
				return nil, err
			}
		}
	}

	ctx := context.Background()
	contacts, err := client.Store.Contacts.GetAllContacts(ctx)
	if err != nil {
		return nil, err
	}

	result := &LabelContactsResult{
		Label:    *label,
		Contacts: make([]LabelContactInfo, 0, len(jids)),
		Synced:   synced,
	}

	seen := make(map[string]int, len(jids))
	for _, rawJID := range jids {
		contact, ok := l.buildLabelContact(ctx, client, contacts, rawJID, data.IncludeGroups)
		if !ok {
			continue
		}

		key := contact.PhoneNumber
		if key == "" {
			key = contact.JID
		}
		if idx, exists := seen[key]; exists {
			if result.Contacts[idx].FullName == "" && contact.FullName != "" {
				result.Contacts[idx] = *contact
			}
			continue
		}
		seen[key] = len(result.Contacts)
		result.Contacts = append(result.Contacts, *contact)
	}

	sort.Slice(result.Contacts, func(i, j int) bool {
		left := contactSortName(result.Contacts[i])
		right := contactSortName(result.Contacts[j])
		if left == right {
			return result.Contacts[i].JID < result.Contacts[j].JID
		}
		return left < right
	})
	result.Total = len(result.Contacts)

	return result, nil
}

func (l *labelService) GetAllContactsByLabel(instance *instance_model.Instance) (*AllLabelContactsResult, error) {
	client, err := l.ensureClientConnected(instance.Id)
	if err != nil {
		return nil, err
	}

	rows, err := l.labelRepository.GetAllLabelChatsByInstanceID(instance.Id)
	if err != nil {
		return nil, err
	}

	synced := false
	if len(rows) == 0 {
		// First use after deploying the feature: pull native label associations once.
		if err := l.syncLabelAppState(client, instance.Id); err != nil {
			return nil, err
		}
		synced = true
		rows, err = l.labelRepository.GetAllLabelChatsByInstanceID(instance.Id)
		if err != nil {
			return nil, err
		}
	}

	ctx := context.Background()
	contacts, err := client.Store.Contacts.GetAllContacts(ctx)
	if err != nil {
		return nil, err
	}

	labelsByJID := make(map[string][]string, len(rows))
	for _, row := range rows {
		name := cleanLabelName(row.LabelName)
		if name == "" {
			continue
		}
		labelsByJID[row.JID] = append(labelsByJID[row.JID], name)
	}

	result := &AllLabelContactsResult{
		Contacts: make([]AllLabelContactInfo, 0, len(labelsByJID)),
		Synced:   synced,
	}
	index := make(map[string]int, len(labelsByJID))
	for rawJID, labelNames := range labelsByJID {
		contact, ok := l.buildLabelContact(ctx, client, contacts, rawJID, false)
		if !ok {
			continue
		}

		key := contact.PhoneNumber
		if key == "" {
			key = contact.JID
		}
		idx, exists := index[key]
		if !exists {
			index[key] = len(result.Contacts)
			result.Contacts = append(result.Contacts, AllLabelContactInfo{
				JID:         contact.JID,
				PhoneNumber: contact.PhoneNumber,
				LID:         contact.LID,
				IsGroup:     contact.IsGroup,
				Labels:      []string{},
			})
			idx = index[key]
		}

		for _, labelName := range labelNames {
			duplicate := false
			for _, existing := range result.Contacts[idx].Labels {
				if normalizeLabelName(existing) == normalizeLabelName(labelName) {
					duplicate = true
					break
				}
			}
			if !duplicate {
				result.Contacts[idx].Labels = append(result.Contacts[idx].Labels, labelName)
			}
		}
	}

	sort.Slice(result.Contacts, func(i, j int) bool {
		return result.Contacts[i].JID < result.Contacts[j].JID
	})
	result.Total = len(result.Contacts)

	return result, nil
}

func contactSortName(contact LabelContactInfo) string {
	for _, value := range []string{contact.FullName, contact.PushName, contact.BusinessName, contact.FirstName, contact.PhoneNumber} {
		if strings.TrimSpace(value) != "" {
			return strings.ToLower(value)
		}
	}
	return ""
}

func (l *labelService) buildLabelContact(
	ctx context.Context,
	client *whatsmeow.Client,
	contacts map[types.JID]types.ContactInfo,
	rawJID string,
	includeGroups bool,
) (*LabelContactInfo, bool) {
	jid, err := types.ParseJID(rawJID)
	if err != nil {
		return nil, false
	}
	jid = jid.ToNonAD()

	isGroup := jid.Server == types.GroupServer
	if isGroup && !includeGroups {
		return nil, false
	}

	contact := &LabelContactInfo{JID: jid.String(), IsGroup: isGroup}
	if isGroup {
		return contact, true
	}

	contactJID := jid
	switch jid.Server {
	case types.DefaultUserServer:
		contact.PhoneNumber = jid.User
		if lid, err := client.Store.LIDs.GetLIDForPN(ctx, jid); err == nil && !lid.IsEmpty() {
			contact.LID = lid.User
		}
	case types.HiddenUserServer:
		contact.LID = jid.User
		if pn, err := client.Store.LIDs.GetPNForLID(ctx, jid); err == nil && !pn.IsEmpty() {
			contactJID = pn.ToNonAD()
			contact.PhoneNumber = contactJID.User
		}
	}

	info, found := contacts[jid]
	if !found && contactJID != jid {
		info, found = contacts[contactJID]
	}
	if !found && contact.LID != "" {
		info, found = contacts[types.NewJID(contact.LID, types.HiddenUserServer)]
	}
	if found {
		contact.FirstName = info.FirstName
		contact.FullName = info.FullName
		contact.PushName = info.PushName
		contact.BusinessName = info.BusinessName
	}

	if contact.PhoneNumber == "" && contactJID.Server == types.DefaultUserServer {
		contact.PhoneNumber = contactJID.User
	}

	return contact, true
}

func NewLabelService(
	clientPointer map[string]*whatsmeow.Client,
	whatsmeowService whatsmeow_service.WhatsmeowService,
	labelRepository label_repository.LabelRepository,
	loggerWrapper *logger_wrapper.LoggerManager,
) LabelService {
	return &labelService{
		clientPointer:    clientPointer,
		whatsmeowService: whatsmeowService,
		labelRepository:  labelRepository,
		loggerWrapper:    loggerWrapper,
	}
}
