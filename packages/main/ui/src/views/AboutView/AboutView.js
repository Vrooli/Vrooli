import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { GitHubIcon, OrganizationIcon, TwitterIcon, WebsiteIcon } from "@local/icons";
import { Box, Button, IconButton, keyframes, Link, Stack, styled, Tooltip, Typography, useTheme } from "@mui/material";
import { TopBar } from "../../components/navigation/TopBar/TopBar";
import { slideTitle, textPop } from "../../styles";
import { openLink, useLocation } from "../../utils/route";
const wave = keyframes `
  0% {
    transform: rotate(0deg);
  }
  10% {
    transform: rotate(30deg);
  }
  20% {
    transform: rotate(0deg);
  }
  100% {
    transform: rotate(0deg);
  }
`;
const RotatedBox = styled("div")({
    display: "inline-block",
    animation: `${wave} 3s infinite ease`,
});
const memberButtonProps = {
    background: "transparent",
    border: "0",
    "&:hover": {
        background: "transparent",
        filter: "brightness(1.2)",
        transform: "scale(1.2)",
    },
    transition: "all 0.2s ease",
};
const teamMembers = [
    {
        fullName: "Matt Halloran",
        role: "Leader/developer",
        photo: "assets/img/profile-matt.jpg",
        socials: {
            website: "https://matthalloran.info",
            twitter: "https://twitter.com/mdhalloran",
            github: "https://github.com/MattHalloran",
        },
    },
];
const joinTeamLink = "https://github.com/Vrooli/Vrooli#-join-the-team";
export const AboutView = ({ display = "page", }) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    return (_jsxs(Box, { ml: 2, mr: 2, children: [_jsx(TopBar, { display: display, onClose: () => { }, titleData: {
                    titleKey: "AboutUs",
                    hideOnDesktop: true,
                } }), _jsxs(Stack, { mt: 4, spacing: 4, children: [_jsxs(Box, { children: [_jsxs(Typography, { variant: "h4", gutterBottom: true, children: ["Hello there! ", _jsx(RotatedBox, { sx: { display: "inline-block" }, children: "\uD83D\uDC4B" })] }), _jsx(Typography, { variant: "body1", children: "Welcome to Vrooli, a platform designed to tackle the challenges of transparency and reliability in autonomous systems. We're passionate about creating a cooperative organizational layer that fosters collaboration between humans and digital actors in a decentralized manner. Our top priority is developing systems that are both ethical and beneficial to society, instead of just chasing profit or power." })] }), _jsxs(Box, { children: [_jsx(Typography, { variant: "h5", gutterBottom: true, children: "The Hypergraph" }), _jsx(Typography, { variant: "body1", children: "At the heart of Vrooli is the concept of a hypergraph of routines. This innovative structure represents an automated economy through interconnected routines, with each routine linked to a specific task or function. By embracing a flexible and adaptable approach to building autonomous systems, we can meet the ever-changing needs of various industries while keeping our focus on transparency and reliability." })] }), _jsxs(Box, { children: [_jsx(Typography, { variant: "h5", gutterBottom: true, children: "Streamlining Organizational Processes" }), _jsx(Typography, { variant: "body1", children: "Leveraging advanced language models, Vrooli will soon be able to generate standards and routines that automate intra- and inter-organizational processes. We stand out from traditional automation tools by allowing processes to be built hierarchically, reducing the complexity of automating a business. For instance, a routine could be created to handle customer support inquiries, which then branches out into subroutines for processing refunds, answering frequently asked questions, or escalating issues to the right team members. With our program, you can group these processes into an organization, making it easy to create your own business by simply copying an organization template and accessing its internal processes." })] }), _jsxs(Box, { children: [_jsx(Typography, { variant: "h5", gutterBottom: true, children: "Integrating Human and Bot Employees" }), _jsx(Typography, { variant: "body1", children: "Vrooli's platform is designed with both human and bot employees in mind, ensuring seamless integration within an organization. We're excited to be launching a messaging feature later this year that allows users to interact with bots for guidance, feedback, and task assignments within your organization. This feature will promote better communication and collaboration between human employees and bots, leading to a more efficient and cohesive work environment." })] }), _jsxs(Box, { children: [_jsx(Typography, { variant: "h5", gutterBottom: true, children: "Reducing Costs" }), _jsx(Typography, { variant: "body1", children: "Starting and maintaining an organization can be expensive, but Vrooli has a solution. Since the hypergraph is publicly defined and shared, humans and bots can collaborate to improve all organizations simultaneously. As organizations become more automated, the cost of running businesses that don't rely heavily on physical labor will drop significantly. These savings can benefit consumers, support business growth and development, or even enhance employee benefits. The upcoming adoption of humanoid robots and autonomous vehicles will further unlock the potential of this system and increase individual productivity." })] }), _jsxs(Box, { children: [_jsx(Typography, { variant: "h5", gutterBottom: true, children: "Our Commitment" }), _jsx(Typography, { variant: "body1", children: "At Vrooli, we're committed to ethical and beneficial outcomes in the development of autonomous systems. By concentrating on responsible and sustainable development, our platform champions a collaborative and distributed approach to promoting transparency and reliability across diverse industries." })] }), _jsxs(Box, { children: [_jsx(Typography, { variant: "h5", gutterBottom: true, children: "Get Involved" }), _jsxs(Typography, { variant: "body1", children: ["We'd love for you to join us in learning more about Vrooli and how our platform can contribute to creating transparent and reliable autonomous systems. Together, we can build a future where technology serves society in the most ethical and transparent way possible. To get started, visit our ", _jsx(Link, { href: "https://github.com/Vrooli/Vrooli#-join-the-team", target: "_blank", rel: "noopener", children: "GitHub page" }), ". We're looking forward to embarking on this exciting journey with you!"] })] })] }), _jsx(Typography, { variant: 'h2', component: "h1", pb: 4, mt: 5, sx: { ...slideTitle }, children: "The Team" }), _jsxs(Stack, { id: "members-stack", direction: "column", spacing: 10, children: [teamMembers.map((member, key) => (_jsxs(Box, { sx: {
                            display: "flex",
                            alignItems: "center",
                            justifyContent: "space-between",
                            height: "300px",
                            boxShadow: 12,
                            backgroundColor: palette.primary.dark,
                            borderRadius: 2,
                            padding: 2,
                            overflow: "overlay",
                        }, children: [_jsx(Box, { component: "img", src: member.photo, alt: `${member.fullName} profile picture`, sx: {
                                    maxWidth: "min(300px, 40%)",
                                    maxHeight: "200px",
                                    objectFit: "contain",
                                    borderRadius: "100%",
                                } }), _jsxs(Box, { sx: {
                                    width: "min(300px, 50%)",
                                    height: "fit-content",
                                }, children: [_jsx(Typography, { variant: 'h4', mb: 1, sx: { ...textPop }, children: member.fullName }), _jsx(Typography, { variant: 'h6', mb: 2, sx: { ...textPop }, children: member.role }), _jsxs(Stack, { direction: "row", alignItems: "center", justifyContent: "center", children: [member.socials.website && (_jsx(Tooltip, { title: "Personal website", placement: "bottom", children: _jsx(IconButton, { onClick: () => openLink(setLocation, member.socials.website), sx: memberButtonProps, children: _jsx(WebsiteIcon, { fill: palette.secondary.light, width: "50px", height: "50px" }) }) })), member.socials.twitter && (_jsx(Tooltip, { title: "Twitter", placement: "bottom", children: _jsx(IconButton, { onClick: () => openLink(setLocation, member.socials.twitter), sx: memberButtonProps, children: _jsx(TwitterIcon, { fill: palette.secondary.light, width: "42px", height: "42px" }) }) })), member.socials.github && (_jsx(Tooltip, { title: "GitHub", placement: "bottom", children: _jsx(IconButton, { onClick: () => openLink(setLocation, member.socials.github), sx: memberButtonProps, children: _jsx(GitHubIcon, { fill: palette.secondary.light, width: "42px", height: "42px" }) }) }))] })] })] }))), _jsx(Stack, { direction: "row", justifyContent: "center", alignItems: "center", pt: 4, spacing: 2, children: _jsx(Button, { size: "large", color: "secondary", href: joinTeamLink, onClick: (e) => {
                                e.preventDefault();
                                openLink(setLocation, joinTeamLink);
                            }, startIcon: _jsx(OrganizationIcon, {}), children: "Join the Team" }) })] })] }));
};
//# sourceMappingURL=AboutView.js.map