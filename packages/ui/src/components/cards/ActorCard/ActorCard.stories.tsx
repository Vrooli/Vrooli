import { Meta, Story } from "@storybook/react";
import { ActorCard as Component } from '../';
import { ActorCardProps as Props } from '../types';

// Define story metadata
export default {
    title: 'cards/ActorCard',
    component: Component,
} as Meta;

// Define template for enabling control over props
const Template: Story<Props> = (args) => <Component {...args} />;

// Export story
export const Default = Template.bind({});