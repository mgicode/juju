// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package controller

import (
	"fmt"
	"net/url"
	"time"

	"github.com/juju/errors"
	"github.com/juju/schema"
	"github.com/juju/utils"
	utilscert "github.com/juju/utils/cert"
	"gopkg.in/juju/names.v2"
	"gopkg.in/macaroon-bakery.v1/bakery"

	"github.com/juju/juju/cert"
)

const (
	// MongoProfLow represents the most conservative mongo memory profile.
	MongoProfLow = "low"
	// MongoProfDefault represents the mongo memory profile shipped by default.
	MongoProfDefault = "default"
)

const (
	// APIPort is the port used for api connections.
	APIPort = "api-port"

	// AuditingEnabled determines whether the controller will record
	// auditing information.
	AuditingEnabled = "auditing-enabled"

	// StatePort is the port used for mongo connections.
	StatePort = "state-port"

	// CACertKey is the key for the controller's CA certificate attribute.
	CACertKey = "ca-cert"

	// ControllerUUIDKey is the key for the controller UUID attribute.
	ControllerUUIDKey = "controller-uuid"

	// IdentityURL sets the url of the identity manager.
	IdentityURL = "identity-url"

	// IdentityPublicKey sets the public key of the identity manager.
	IdentityPublicKey = "identity-public-key"

	// SetNUMAControlPolicyKey stores the value for this setting
	SetNUMAControlPolicyKey = "set-numa-control-policy"

	// AutocertDNSNameKey sets the DNS name of the controller. If a
	// client connects to this name, an official certificate will be
	// automatically requested. Connecting to any other host name
	// will use the usual self-generated certificate.
	AutocertDNSNameKey = "autocert-dns-name"

	// AutocertURLKey sets the URL used to obtain official TLS
	// certificates when a client connects to the API. By default,
	// certficates are obtains from LetsEncrypt. A good value for
	// testing is
	// "https://acme-staging.api.letsencrypt.org/directory".
	AutocertURLKey = "autocert-url"

	// AllowModelAccessKey sets whether the controller will allow users to
	// connect to models they have been authorized for even when
	// they don't have any access rights to the controller itself.
	AllowModelAccessKey = "allow-model-access"

	// MongoMemoryProfile sets whether mongo uses the least possible memory or the
	// detault
	MongoMemoryProfile = "mongo-memory-profile"

	// MaxLogsAge is the maximum age for log entries, ef "72h"
	MaxLogsAge = "max-logs-age"

	// MaxLogsSize is the maximum size the log collection can grow to
	// before it is pruned, eg "4M"
	MaxLogsSize = "max-logs-size"

	// MaxTxnLogSize is the maximum size the of capped txn log collection, eg "10M"
	MaxTxnLogSize = "max-txn-log-size"

	// Attribute Defaults

	// DefaultAuditingEnabled contains the default value for the
	// AuditingEnabled config value.
	DefaultAuditingEnabled = false

	// DefaultNUMAControlPolicy should not be used by default.
	// Only use numactl if user specifically requests it
	DefaultNUMAControlPolicy = false

	// DefaultStatePort is the default port the controller is listening on.
	DefaultStatePort int = 37017

	// DefaultAPIPort is the default port the API server is listening on.
	DefaultAPIPort int = 17070

	// DefaultMongoMemoryProfile is the default profile used by mongo.
	DefaultMongoMemoryProfile = MongoProfLow

	// DefaultMaxLogsAgeDays is the maximum age in days of log entries.
	DefaultMaxLogsAgeDays = 3

	// DefaultMaxLogCollectionMB is the maximum size the log collection can
	// grow to before being pruned.
	DefaultMaxLogCollectionMB = 4 * 1024 // 4 GB

	// DefaultMaxTxnLogCollectionMB is the maximum size the txn log collection.
	DefaultMaxTxnLogCollectionMB = 10 // 10 MB

	// JujuHASpace is the network space within which the MongoDB replica-set
	// should communicate.
	JujuHASpace = "juju-ha-space"

	// JujuManagementSpace is the network space that agents should use to
	// communicate with controllers.
	JujuManagementSpace = "juju-mgmt-space"
)

// ControllerOnlyConfigAttributes are attributes which are only relevant
// for a controller, never a model.
var ControllerOnlyConfigAttributes = []string{
	AllowModelAccessKey,
	APIPort,
	AutocertDNSNameKey,
	AutocertURLKey,
	CACertKey,
	ControllerUUIDKey,
	IdentityPublicKey,
	IdentityURL,
	SetNUMAControlPolicyKey,
	StatePort,
	MongoMemoryProfile,
	MaxLogsSize,
	MaxLogsAge,
	MaxTxnLogSize,
	JujuHASpace,
	JujuManagementSpace,
}

// ControllerOnlyAttribute returns true if the specified attribute name
// is only relevant for a controller.
func ControllerOnlyAttribute(attr string) bool {
	for _, a := range ControllerOnlyConfigAttributes {
		if attr == a {
			return true
		}
	}
	return false
}

// Config is a string-keyed map of controller configuration attributes.
type Config map[string]interface{}

// Validate validates the controller configuration.
func (c Config) Validate() error {
	return Validate(c)
}

// NewConfig creates a new Config from the supplied attributes.
// Default values will be used where defaults are available.
//
// The controller UUID and CA certificate must be passed in.
// The UUID is typically generated by the immediate caller,
// and the CA certificate generated by environs/bootstrap.NewConfig.
func NewConfig(controllerUUID, caCert string, attrs map[string]interface{}) (Config, error) {
	coerced, err := configChecker.Coerce(attrs, nil)
	if err != nil {
		return Config{}, errors.Trace(err)
	}
	attrs = coerced.(map[string]interface{})
	attrs[ControllerUUIDKey] = controllerUUID
	attrs[CACertKey] = caCert
	config := Config(attrs)
	return config, config.Validate()
}

// mustInt returns the named attribute as an integer, panicking if
// it is not found or is zero. Zero values should have been
// diagnosed at Validate time.
func (c Config) mustInt(name string) int {
	// Values obtained over the api are encoded as float64.
	if value, ok := c[name].(float64); ok {
		return int(value)
	}
	value, _ := c[name].(int)
	if value == 0 {
		panic(errors.Errorf("empty value for %q found in configuration", name))
	}
	return value
}

// asString is a private helper method to keep the ugly string casting
// in once place. It returns the given named attribute as a string,
// returning "" if it isn't found.
func (c Config) asString(name string) string {
	value, _ := c[name].(string)
	return value
}

// mustString returns the named attribute as an string, panicking if
// it is not found or is empty.
func (c Config) mustString(name string) string {
	value, _ := c[name].(string)
	if value == "" {
		panic(errors.Errorf("empty value for %q found in configuration (type %T, val %v)", name, c[name], c[name]))
	}
	return value
}

// StatePort returns the controller port for the environment.
func (c Config) StatePort() int {
	return c.mustInt(StatePort)
}

// APIPort returns the API server port for the environment.
func (c Config) APIPort() int {
	return c.mustInt(APIPort)
}

// AuditingEnabled returns whether or not auditing has been enabled
// for the environment. The default is false.
func (c Config) AuditingEnabled() bool {
	if v, ok := c[AuditingEnabled]; ok {
		return v.(bool)
	}
	return false
}

// ControllerUUID returns the uuid for the model's controller.
func (c Config) ControllerUUID() string {
	return c.mustString(ControllerUUIDKey)
}

// CACert returns the certificate of the CA that signed the controller
// certificate, in PEM format, and whether the setting is available.
//
// TODO(axw) once the controller config is completely constructed,
// there will always be a CA certificate. Get rid of the bool result.
func (c Config) CACert() (string, bool) {
	if s, ok := c[CACertKey]; ok {
		return s.(string), true
	}
	return "", false
}

// IdentityURL returns the url of the identity manager.
func (c Config) IdentityURL() string {
	return c.asString(IdentityURL)
}

// AutocertURL returns the URL used to obtain official TLS certificates
// when a client connects to the API. See AutocertURLKey
// for more details.
func (c Config) AutocertURL() string {
	return c.asString(AutocertURLKey)
}

// AutocertDNSName returns the DNS name of the controller.
// See AutocertDNSNameKey for more details.
func (c Config) AutocertDNSName() string {
	return c.asString(AutocertDNSNameKey)
}

// IdentityPublicKey returns the public key of the identity manager.
func (c Config) IdentityPublicKey() *bakery.PublicKey {
	key := c.asString(IdentityPublicKey)
	if key == "" {
		return nil
	}
	var pubKey bakery.PublicKey
	err := pubKey.UnmarshalText([]byte(key))
	if err != nil {
		// We check if the key string can be unmarshalled into a PublicKey in the
		// Validate function, so we really do not expect this to fail.
		panic(err)
	}
	return &pubKey
}

// MongoMemoryProfile returns the selected profile or low.
func (c Config) MongoMemoryProfile() string {
	if profile, ok := c[MongoMemoryProfile]; ok {
		return profile.(string)
	}
	return MongoProfLow
}

// NUMACtlPreference returns if numactl is preferred.
func (c Config) NUMACtlPreference() bool {
	if numa, ok := c[SetNUMAControlPolicyKey]; ok {
		return numa.(bool)
	}
	return DefaultNUMAControlPolicy
}

// AllowModelAccess reports whether users are allowed to access models
// they have been granted permission for even when they can't access
// the controller.
func (c Config) AllowModelAccess() bool {
	value, _ := c[AllowModelAccessKey].(bool)
	return value
}

// MaxLogsAge is the maximum age of log entries before they are pruned.
func (c Config) MaxLogsAge() time.Duration {
	// Value has already been validated.
	val, _ := time.ParseDuration(c.mustString(MaxLogsAge))
	return val
}

// MaxLogSizeMB is the maximum size in MiB which the log collection
// can grow to before being pruned.
func (c Config) MaxLogSizeMB() int {
	// Value has already been validated.
	val, _ := utils.ParseSize(c.mustString(MaxLogsSize))
	return int(val)
}

// MaxTxnLogSizeMB is the maximum size in MiB of the txn log collection.
func (c Config) MaxTxnLogSizeMB() int {
	// Value has already been validated.
	val, _ := utils.ParseSize(c.mustString(MaxTxnLogSize))
	return int(val)
}

// JujuHASpace is the network space within which the MongoDB replica-set
// should communicate.
func (c Config) JujuHASpace() string {
	return c.asString(JujuHASpace)
}

// JujuManagementSpace is the network space that agents should use to
// communicate with controllers.
func (c Config) JujuManagementSpace() string {
	return c.asString(JujuManagementSpace)
}

// Validate ensures that config is a valid configuration.
func Validate(c Config) error {
	if v, ok := c[IdentityPublicKey].(string); ok {
		var key bakery.PublicKey
		if err := key.UnmarshalText([]byte(v)); err != nil {
			return errors.Annotate(err, "invalid identity public key")
		}
	}

	if v, ok := c[IdentityURL].(string); ok {
		u, err := url.Parse(v)
		if err != nil {
			return errors.Annotate(err, "invalid identity URL")
		}
		// If we've got an identity public key, we allow an HTTP
		// scheme for the identity server because we won't need
		// to rely on insecure transport to obtain the public
		// key.
		if _, ok := c[IdentityPublicKey]; !ok && u.Scheme != "https" {
			return errors.Errorf("URL needs to be https when %s not provided", IdentityPublicKey)
		}
	}

	caCert, caCertOK := c.CACert()
	if !caCertOK {
		return errors.Errorf("missing CA certificate")
	}
	if _, err := utilscert.ParseCert(caCert); err != nil {
		return errors.Annotate(err, "bad CA certificate in configuration")
	}

	if uuid, ok := c[ControllerUUIDKey].(string); ok && !utils.IsValidUUIDString(uuid) {
		return errors.Errorf("controller-uuid: expected UUID, got string(%q)", uuid)
	}

	if mgoMemProfile, ok := c[MongoMemoryProfile].(string); ok {
		if mgoMemProfile != MongoProfLow && mgoMemProfile != MongoProfDefault {
			return errors.Errorf("mongo-memory-profile: expected one of %s or %s got string(%q)", MongoProfLow, MongoProfDefault, mgoMemProfile)
		}
	}

	if v, ok := c[MaxLogsAge].(string); ok {
		if _, err := time.ParseDuration(v); err != nil {
			return errors.Annotate(err, "invalid logs prune interval in configuration")
		}
	}

	if v, ok := c[MaxLogsSize].(string); ok {
		if _, err := utils.ParseSize(v); err != nil {
			return errors.Annotate(err, "invalid max logs size in configuration")
		}
	}

	if v, ok := c[MaxTxnLogSize].(string); ok {
		if _, err := utils.ParseSize(v); err != nil {
			return errors.Annotate(err, "invalid max txn log size in configuration")
		}
	}

	if err := validateSpaceConfig(c, JujuHASpace, "juju HA"); err != nil {
		return errors.Trace(err)
	}

	if err := validateSpaceConfig(c, JujuManagementSpace, "juju mgmt"); err != nil {
		return errors.Trace(err)
	}

	return nil
}

func validateSpaceConfig(c Config, key, topic string) error {
	val := c[key]
	if val == nil {
		return nil
	}
	if v, ok := val.(string); ok {
		if !names.IsValidSpace(v) {
			return errors.NotValidf("%s space name %q", topic, val)
		}
	} else {
		return errors.NotValidf("type for %s space name %v", topic, val)
	}

	return nil
}

// GenerateControllerCertAndKey makes sure that the config has a CACert and
// CAPrivateKey, generates and returns new certificate and key.
func GenerateControllerCertAndKey(caCert, caKey string, hostAddresses []string) (string, string, error) {
	return cert.NewDefaultServer(caCert, caKey, hostAddresses)
}

var configChecker = schema.FieldMap(schema.Fields{
	AuditingEnabled:         schema.Bool(),
	APIPort:                 schema.ForceInt(),
	StatePort:               schema.ForceInt(),
	IdentityURL:             schema.String(),
	IdentityPublicKey:       schema.String(),
	SetNUMAControlPolicyKey: schema.Bool(),
	AutocertURLKey:          schema.String(),
	AutocertDNSNameKey:      schema.String(),
	AllowModelAccessKey:     schema.Bool(),
	MongoMemoryProfile:      schema.String(),
	MaxLogsAge:              schema.String(),
	MaxLogsSize:             schema.String(),
	MaxTxnLogSize:           schema.String(),
	JujuHASpace:             schema.String(),
	JujuManagementSpace:     schema.String(),
}, schema.Defaults{
	APIPort:                 DefaultAPIPort,
	AuditingEnabled:         DefaultAuditingEnabled,
	StatePort:               DefaultStatePort,
	IdentityURL:             schema.Omit,
	IdentityPublicKey:       schema.Omit,
	SetNUMAControlPolicyKey: DefaultNUMAControlPolicy,
	AutocertURLKey:          schema.Omit,
	AutocertDNSNameKey:      schema.Omit,
	AllowModelAccessKey:     schema.Omit,
	MongoMemoryProfile:      schema.Omit,
	MaxLogsAge:              fmt.Sprintf("%vh", DefaultMaxLogsAgeDays*24),
	MaxLogsSize:             fmt.Sprintf("%vM", DefaultMaxLogCollectionMB),
	MaxTxnLogSize:           fmt.Sprintf("%vM", DefaultMaxTxnLogCollectionMB),
	JujuHASpace:             schema.Omit,
	JujuManagementSpace:     schema.Omit,
})
