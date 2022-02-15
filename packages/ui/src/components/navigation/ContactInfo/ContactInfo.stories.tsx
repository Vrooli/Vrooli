import { Meta, Story } from "@storybook/react";
import { ContactInfo as Component } from '../';
import { ContactInfoProps as Props } from '../types';

// Define story metadata
export default {
    title: 'navigation/ContactInfo',
    component: Component,
} as Meta;

// Define template for enabling control over props
const Template: Story<Props> = (args) => <Component {...args} />;

// Export story
export const Default = Template.bind({});