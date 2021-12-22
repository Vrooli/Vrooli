import { Meta, Story } from "@storybook/react";
import { RoutineCard as Component } from '../';
import { RoutineCardProps as Props } from '../types';

// Define story metadata
export default {
    title: 'cards/RoutineCard',
    component: Component,
} as Meta;

// Define template for enabling control over props
const Template: Story<Props> = (args) => <Component {...args} />;

// Export story
export const Default = Template.bind({});