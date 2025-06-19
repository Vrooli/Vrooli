/**
 * Form Data Fixtures Index
 * 
 * This module exports all form data fixtures for UI testing.
 * Form data fixtures represent data as it would appear in form state
 * before submission to APIs.
 * 
 * These fixtures are specifically designed for:
 * - Testing form components and validation
 * - Testing user input scenarios
 * - Testing form state management
 * - Testing form-to-API data transformation
 * 
 * For API response testing, use fixtures from @vrooli/shared/__test/fixtures/api
 */

// Export all form data fixtures
export * from './apiKeyFormData.js';
export * from './apiKeyExternalFormData.js';
export * from './bookmarkFormData.js';
export * from './bookmarkListFormData.js';
export * from './botFormData.js';
export * from './chatFormData.js';
export * from './chatInviteFormData.js';
export * from './chatMessageFormData.js';
export * from './chatParticipantFormData.js';
export * from './commentFormData.js';
export * from './emailFormData.js';
export * from './issueFormData.js';
export * from './meetingFormData.js';
export * from './meetingInviteFormData.js';
export * from './memberFormData.js';
export * from './memberInviteFormData.js';
export * from './notificationSubscriptionFormData.js';
export * from './phoneFormData.js';
export * from './projectFormData.js';
export * from './pullRequestFormData.js';
export * from './pushDeviceFormData.js';
export * from './reminderFormData.js';
export * from './reminderItemFormData.js';
export * from './reminderListFormData.js';
export * from './reportFormData.js';
export * from './reportResponseFormData.js';
export * from './resourceFormData.js';
export * from './resourceVersionFormData.js';
export * from './resourceVersionRelationFormData.js';
export * from './runFormData.js';
export * from './runIOFormData.js';
export * from './runStepFormData.js';
export * from './scheduleFormData.js';
export * from './scheduleExceptionFormData.js';
export * from './scheduleRecurrenceFormData.js';
export * from './tagFormData.js';
export * from './teamFormData.js';
export * from './transferFormData.js';
export * from './userFormData.js';
export * from './walletFormData.js';

/**
 * Complete collection of all form data fixtures organized by object type.
 * Use this for importing all fixtures at once or for comprehensive testing.
 */
export const formDataFixtures = {
    // Re-export all fixtures here for convenient access
    // Import will be done dynamically to avoid circular dependencies
};