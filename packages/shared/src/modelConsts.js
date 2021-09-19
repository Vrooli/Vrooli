export const ACCOUNT_STATUS = {
    Deleted: 'Deleted',
    Unlocked: 'Unlocked',
    SoftLock: 'SoftLock',
    HardLock: 'HardLock'
}
export const DEFAULT_PRONOUNS = [
    "he/him/his",
    "she/her/hers",
    "they/them/theirs",
    "ze/zir/zirs",
    "ze/hir/hirs",
]
export const IMAGE_EXTENSION = {
    Bmp: '.bmp',
    Gif: '.gif',
    Png: '.png',
    Jpg: '.jpg',
    Jpeg: '.jpeg',
    Heic: '.heic',
    Heif: '.heif',
    Ico: '.ico'
}
// Possible image sizes stored, and their max size
export const IMAGE_SIZE = {
    XXS: 32,
    XS: 64,
    S: 128,
    M: 256,
    ML: 512,
    L: 1024,
    XL: 2048,
    XXL: 4096,
}
export const IMAGE_USE = {
    Hero: 'Hero',
    Gallery: 'Gallery',
    ProductDisplay: 'Display'
}
// CANCELED_BY_ADMIN    | Admin canceled the order at any point before delivery
// CANCELED_BY_CUSTOMER |   1) Customer canceled order before approval (i.e. no admin approval needed), OR
//                          2) PENDING_CANCEL was approved by admin
// PENDING_CANCEL       | Customer canceled order after approval (i.e. admin approval needed)
// REJECTED             | Order was pending, but admin denied it
// DRAFT                | Order that hasn't been submitted yet (i.e. cart)
// PENDING              | Order that has been submitted, but not approved by admin yet
// APPROVED             | Order that has been approved by admin
// SCHEDULED            | Order has been scheduled for delivery
// IN_TRANSIT           | Order is currently being delivered
// DELIVERED            | Order has been delivered
export const ORDER_STATUS = {
    CanceledByAdmin: 'CanceledByAdmin',
    CanceledByCustomer: 'CanceledByCustomer',
    PendingCancel: 'PendingCancel',
    Rejected: 'Rejected',
    Draft: 'Draft',
    Pending: 'Pending',
    Approved: 'Approved',
    Scheduled: 'Scheduled',
    InTransit: 'InTransit',
    Delivered: 'Delivered'
}
export const PRODUCT_SORT_OPTIONS = {
    AZ: 'AZ',
    ZA: 'ZA',
    Featured: 'Featured',
    Newest: 'Newest',
    Oldest: 'Oldest'
}
export const SKU_SORT_OPTIONS = {
    AZ: 'AZ',
    ZA: 'ZA',
    PriceLowHigh: 'PriceLowHigh',
    PriceHighLow: 'PriceHighLow',
    Featured: 'Featured',
    Newest: 'Newest',
    Oldest: 'Oldest'
}
export const SKU_STATUS = {
    Deleted: 'Deleted',
    Inactive: 'Inactive',
    Active: 'Active',
}
export const TASK_STATUS = {
    Unknown: 'Unknown',
    Failed: 'Failed',
    Active: 'Active',
    Completed: 'Completed',
}
export const THEME = {
    Light: 'light',
    Dark: 'dark'
}
export const TRAIT_NAME = {
    DroughtTolerance: 'Drought tolerance',
    GrownHeight: 'Grown height',
    GrownSpread: 'Grown spread',
    GrowthRate: 'Growth rate',
    JerseryNative: 'Jersey native',
    OptimalLight: 'Optimal light',
    ProductType: 'Ship type',
    SaltTolerance: 'Salt tolerance',
    AttractsPollinatorsAndWildlife: 'Attracts pollinators and wildlife',
    BloomTime: 'Bloom time',
    BloomColor: 'Bloom color',
    Zone: 'Zone',
    PhysiographicRegion: 'Physiographic region',
    SoilMoisture: 'Soil moisture',
    SoilPh: 'Soil PH',
    SoilType: 'Soil Type',
    LightRange: 'Light range'
}
export const ROLES = {
    Customer: "Customer",
    Owner: "Owner",
    Admin: "Admin",
}