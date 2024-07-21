package parser

import (
	"fmt"
	"strings"
	"time"

	"github.com/Velocidex/ordereddict"
)

func (self *EntryObject) Debug() {
	fmt.Println(self.DebugString())
}

// Gets the log entry and parse the fields according to documentation
// at
// https://www.freedesktop.org/software/systemd/man/latest/systemd.journal-fields.html
func (self *EntryObject) GetParsed(ctx *JournalFile, size int64) *ordereddict.Dict {
	// This contains system specified fields that can not be changed
	// by the logging client (all fields with leading _)
	system := ordereddict.NewDict().Set("Seq", self.seqnum())

	// Event specific data set by the logging client.
	event_data := ordereddict.NewDict()

	result := ordereddict.NewDict().
		Set("System", system).
		Set("EventData", event_data)

	// Add timestamps and such
	system.Set("Timestamp", time.Unix(0, self.realtime()*1000))

	for idx, item := range self.items(ctx, size) {
		parts := strings.SplitN(item, "=", 2)
		// Should not really happen but we capture it anyway
		if len(parts) < 2 {
			event_data.Set(fmt.Sprintf("Item%d", idx), item)
			continue
		}

		field := parts[0]
		value := parts[1]
		switch field {
		case "ERRONO", "CODE_LINE", "SYSLOG_FACILITY", "SYSLOG_PID", "TID":
			int_val, err := ToInt64(value)
			if err == nil {
				event_data.Set(field, int_val)
				continue
			}

		case "_PID", "_PPID", "_UID", "_EUID", "_FSUID", "_EGID",
			"_GID", "_AUDIT_LOGINUID", "_AUDIT_SESSION",
			"_AUDIT_TYPE", "_AUDIT_ID", "_AUDIT_FIELD_SYSCALL",
			"_AUDIT_FIELD_EXIT", "_AUDIT_FIELD_SUID",
			"_AUDIT_FIELD_SGID":
			int_val, err := ToInt64(value)
			if err == nil {
				system.Set(field, int_val)
				continue
			}

		case "SYSLOG_TIMESTAMP":
			int_val, err := ToInt64(value)
			if err == nil {
				event_data.Set(field, time.Unix(int_val, 0))
				continue
			}

		case "_SOURCE_REALTIME_TIMESTAMP":
			int_val, err := ToInt64(value)
			if err == nil {
				system.Set(field[1:], time.Unix(0, int_val*1000))
				continue
			}

		case "PRIORITY":
			priority, err := ToInt64(value)
			if err == nil {
				event_data.Set(field, priority)
				continue
			}

		default:
		}

		if strings.HasPrefix(field, "_") {
			system.Set(field, value)
		} else {
			event_data.Set(field, value)
		}
	}

	return result
}

// Gets the raw log entry without interpreting the data.
func (self *EntryObject) GetRaw(ctx *JournalFile, size int64) *ordereddict.Dict {
	result := ordereddict.NewDict()

	// Add timestamps and such
	result.Set("Timestamp", time.Unix(0, self.realtime()*1000))

	for idx, item := range self.items(ctx, size) {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) < 2 {
			result.Set(fmt.Sprintf("Item%d", idx), item)
		} else {
			result.Set(parts[0], parts[1])
		}
	}

	return result
}

func (self *EntryObject) items_compact(ctx *JournalFile, size int64) []string {
	var res []string

	i := int64(self.Size())

	for i < size {
		item := self.Profile.CompatEntryItem(self.Reader, i+self.Offset)
		obj_offset := int64(item.object())
		if obj_offset <= 0 || obj_offset > ctx.ArenaSize {
			break
		}

		data_obj := self.Profile.ObjectHeader(self.Reader, obj_offset)
		if data_obj.Type().Name == "OBJECT_DATA" {

			// The payload goes to the end of the object.
			obj_size := data_obj.__real_size()

			// Payload length: less the object header and the
			// offset within the CompatDataObject
			payload_len := obj_size - int64(data_obj.Size()) -
				self.Profile.Off_CompatDataObject_payload

			// Start reading after the object header
			payload := ParseString(self.Reader,
				obj_offset+int64(data_obj.Size())+
					self.Profile.Off_CompatDataObject_payload,
				payload_len)

			res = append(res, payload)
		}
		i += int64(item.Size())
	}

	return res
}

func (self *EntryObject) items_regular(ctx *JournalFile, size int64) []string {
	var res []string

	i := int64(self.Size())

	for i < size {
		item := self.Profile.EntryItem(self.Reader, i+self.Offset)
		obj_offset := int64(item.object())
		if obj_offset <= 0 || obj_offset > ctx.ArenaSize {
			break
		}

		data_obj := self.Profile.ObjectHeader(self.Reader, obj_offset)
		if data_obj.Type().Name == "OBJECT_DATA" {

			// The payload goes to the end of the object.
			obj_size := data_obj.__real_size()

			// Payload length: less the object header and the
			// offset within the CompatDataObject
			payload_len := obj_size - int64(data_obj.Size()) -
				self.Profile.Off_DataObject_payload

			// Start reading after the object header
			payload := ParseString(self.Reader,
				obj_offset+int64(data_obj.Size())+
					self.Profile.Off_DataObject_payload,
				payload_len)

			res = append(res, payload)
		}
		i += int64(item.Size())
	}

	return res
}

func (self *EntryObject) items(ctx *JournalFile, size int64) []string {
	var res []string

	if ctx.IsCompact {
		return self.items_compact(ctx, size)

	} else {
		return self.items_regular(ctx, size)
	}

	return res
}
