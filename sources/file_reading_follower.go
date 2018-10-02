package sources

import (
	log "github.com/sirupsen/logrus"
	"github.com/Pirionfr/lookatch-common/control"
	"github.com/Pirionfr/lookatch-common/events"
	"github.com/papertrail/go-tail/follower"
	"github.com/Pirionfr/lookatch-common/util"
	"io"
	"strconv"
	"time"
	"encoding/json"
	"sync"
)

// FileReadingFollower representation of FileReadingFollower
type FileReadingFollower struct {
	*Source
	config     FileReadingFollowerConfig
	status     string
}

// SyslogConfig representation of FileReadingFollower Config
type FileReadingFollowerConfig struct {
	Path    string   `json:"path"`
	Offset  int64    `json:"offset"`
}


// FileReadingFollowerType type of source
const FileReadingFollowerType = "fileReadingFollower"

// create new FileReadingFollower source
func newFileReadingFollower(s *Source) (SourceI, error) {

	fileReadingFollowerConfig := FileReadingFollowerConfig{}
	s.Conf.UnmarshalKey("sources."+s.Name, &fileReadingFollowerConfig)

	return &FileReadingFollower{
		Source: s,
		config: fileReadingFollowerConfig,
	}, nil
}

// Init source
func (f *FileReadingFollower) Init() {

}

// Stop source
func (f *FileReadingFollower) Stop() error {
	return nil
}

// Start source
func (f *FileReadingFollower) Start(i ...interface{}) error {
	if !util.IsStandalone(f.Conf) {
		var wg sync.WaitGroup
		wg.Add(1)
		//wait for changeStatus
		go func() {
			for f.status == control.SourceStatusWaitingForMETA {
				time.Sleep(time.Second)
			}
			wg.Done()
		}()
		wg.Wait()
	} else {
		f.status = control.SourceStatusRunning
	}
	go f.read()
	return nil
}

// GetName get source name
func (f *FileReadingFollower) GetName() string {
	return f.Name
}

// GetOutputChan get output channel
func (f *FileReadingFollower) GetOutputChan() chan *events.LookatchEvent {
	return f.OutputChannel
}

// IsEnable check if source is enable
func (f *FileReadingFollower) IsEnable() bool {
	return true
}

// HealthCheck return true if ok
func (f *FileReadingFollower) HealthCheck() bool {
	return f.status == control.SourceStatusRunning
}

// GetMeta get source meta
func (f *FileReadingFollower) GetMeta() map[string]interface{} {
	meta := make(map[string]interface{})
	if f.status != control.SourceStatusWaitingForMETA {
		meta["offset"] = f.config.Offset
		meta["offset_agent"] = f.Offset
	}
	return meta
}

// GetSchema Get source Schema
func (f *FileReadingFollower) GetSchema() interface{} {
	return "String"
}

// GetStatus Get source status
func (f *FileReadingFollower) GetStatus() interface{} {
	return f.status
}

// GetAvailableActions returns available actions
func (f *FileReadingFollower) GetAvailableActions() map[string]*control.ActionDescription {
	return nil
}

// Process action
func (f *FileReadingFollower) Process(action string, params ...interface{}) interface{} {
	switch action {
	case control.SourceMeta:
		meta := &control.Meta{}
		payload := params[0].([]byte)
		err := json.Unmarshal(payload, meta)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Fatal("Unable to unmarshal MySQL Query Statement event")
		} else {
			if val, ok := meta.Data["offset"].(string); ok {
				f.config.Offset, _ = strconv.ParseInt(val, 10, 64)
			}

			if val, ok := meta.Data["offset_agent"].(string); ok {
				f.Offset, _ = strconv.ParseInt(val, 10, 64)
			}

			f.status = control.SourceStatusRunning
		}
		break
	default:
		log.WithFields(log.Fields{
			"action": action,
		}).Error("action not implemented")
	}
	return nil
}


func (f *FileReadingFollower) read(){
	t, err := follower.New(f.config.Path, follower.Config{
		Whence: io.SeekCurrent,
		Offset: f.config.Offset,
		Reopen: true,
	})
	if err != nil {
		log.WithError(err).Error("Error while start reader")
	}

	for line := range t.Lines() {

		f.OutputChannel <- &events.LookatchEvent{
			Header: &events.LookatchHeader{
				EventType: FileReadingFollowerType,
			},
			Payload: &events.GenericEvent{
				Tenant:      f.AgentInfo.tenant.Id,
				AgentId:     f.AgentInfo.uuid,
				Timestamp:   strconv.Itoa(int(time.Now().Unix())),
				Environment: f.AgentInfo.tenant.Env,
				Value:       line.String(),
			},
		}
		f.config.Offset++
	}

	if t.Err() != nil {
		log.WithError(t.Err()).Error("Error while reading File")
	}
}