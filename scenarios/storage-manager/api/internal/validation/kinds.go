package validation

import corestorage "github.com/vrooli/api-core/storage"

var scenarioOwnerKinds = []corestorage.OwnerKind{corestorage.OwnerScenario}

func (storageEntryConformance) Kinds() []corestorage.OwnerKind { return nil }
func (crossPlatform) Kinds() []corestorage.OwnerKind           { return nil }

func (backupTargetMissing) Kinds() []corestorage.OwnerKind { return scenarioOwnerKinds }
func (migrationDebt) Kinds() []corestorage.OwnerKind       { return scenarioOwnerKinds }

func (isoRoutedSeams) Kinds() []corestorage.OwnerKind     { return scenarioOwnerKinds }
func (isoFileRoutedSeams) Kinds() []corestorage.OwnerKind { return scenarioOwnerKinds }
func (isoUnverified) Kinds() []corestorage.OwnerKind      { return scenarioOwnerKinds }
func (isoNamespace) Kinds() []corestorage.OwnerKind       { return scenarioOwnerKinds }

func (schemaHasAlter) Kinds() []corestorage.OwnerKind       { return scenarioOwnerKinds }
func (schemaNotIdempotent) Kinds() []corestorage.OwnerKind  { return scenarioOwnerKinds }
func (schemaCentralized) Kinds() []corestorage.OwnerKind    { return scenarioOwnerKinds }
func (schemaNotPerDomain) Kinds() []corestorage.OwnerKind   { return scenarioOwnerKinds }
func (schemaEnsureNotWired) Kinds() []corestorage.OwnerKind { return scenarioOwnerKinds }
func (schemaSystemNotEmpty) Kinds() []corestorage.OwnerKind { return scenarioOwnerKinds }
func (schemaCrossDomainFK) Kinds() []corestorage.OwnerKind  { return scenarioOwnerKinds }

func (hygieneSQLitePoolDeadlock) Kinds() []corestorage.OwnerKind { return scenarioOwnerKinds }
func (hygieneRowsClose) Kinds() []corestorage.OwnerKind          { return scenarioOwnerKinds }
func (hygieneHandleCapture) Kinds() []corestorage.OwnerKind      { return scenarioOwnerKinds }
func (hygieneRoutedDriver) Kinds() []corestorage.OwnerKind       { return scenarioOwnerKinds }
func (hygieneSQLInHandlers) Kinds() []corestorage.OwnerKind      { return scenarioOwnerKinds }
func (hygieneRawSQLOpen) Kinds() []corestorage.OwnerKind         { return scenarioOwnerKinds }
